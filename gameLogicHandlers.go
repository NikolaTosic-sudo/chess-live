package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (cfg *appConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentPieceName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}
	err = r.ParseForm()

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	multiplier, err := strconv.Atoi(r.FormValue("multiplier"))

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return
	}
	currentGame := c.Value
	onlineGame, found := cfg.connections[currentGame]
	match := cfg.Matches[currentGame]
	currentPiece := match.pieces[currentPieceName]
	userC, err := r.Cookie("access_token")

	var userId uuid.UUID
	if err == nil && userC.Value != "" {
		userId, _ = auth.ValidateJWT(userC.Value, cfg.secret)
	}

	canPlay, err := canPlay(currentPiece, match, onlineGame.players, userId)
	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
		return
	}
	currentSquareName := currentPiece.Tile
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile
	selSq := match.board[selectedSquare]
	legalMoves := checkLegalMoves(match)

	if canEat(match.selectedPiece, currentPiece) && slices.Contains(legalMoves, currentSquareName) {
		if found {
			userC, err := r.Cookie("access_token")

			if err != nil {
				respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
				return
			}

			userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

			if err != nil {
				respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
				return
			}

			if match.isWhiteTurn && onlineGame.players["white"].ID != userId {
				return
			} else if !match.isWhiteTurn && onlineGame.players["black"].ID != userId {
				return
			}
		}
		var kingCheck bool
		if match.selectedPiece.IsKing {
			kingCheck = handleChecksWhenKingMoves(currentSquareName, match)
		} else if match.isWhiteTurn && match.isWhiteUnderCheck && !slices.Contains(match.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		} else if !match.isWhiteTurn && match.isBlackUnderCheck && !slices.Contains(match.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var check bool
		if !match.selectedPiece.IsKing {
			check, _, _ = cfg.handleCheckForCheck(currentSquareName, currentGame, match.selectedPiece)
		}

		if check || kingCheck {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var userColor string
		if match.isWhiteTurn {
			match.takenPiecesWhite = append(match.takenPiecesWhite, currentPiece.Image)
			userColor = "white"
		} else {
			match.takenPiecesBlack = append(match.takenPiecesBlack, currentPiece.Image)
			userColor = "black"
		}

		message := fmt.Sprintf(
			getEatPiecesMessage(),
			currentPiece.Name,
			currentPiece.Image,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
			userColor,
			currentPiece.Image,
		)

		err = sendMessage(onlineGame, found, w, message, [2][]int{
			{currentSquare.CoordinatePosition[0]},
			{currentSquare.CoordinatePosition[1]},
		})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't print to page", err)
			return
		}

		match, _, saveSelected := eatCleanup(match, currentPiece, selectedSquare, currentSquareName)

		cfg.Matches[currentGame] = match
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}
		pawnPromotion, err := cfg.checkForPawnPromotion(saveSelected.Name, currentGame, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "pawn promotion error: ", err)
			return
		}

		if saveSelected.IsPawn && pawnPromotion {
			return
		}

		noCheck, err := handleIfCheck(w, r, cfg, saveSelected, currentGame)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "handle check error: ", err)
			return
		}
		if noCheck {
			var kingName string
			if match.isWhiteUnderCheck {
				kingName = "white_king"
			} else if match.isBlackUnderCheck {
				kingName = "black_king"
			} else {
				cfg.endTurn(currentGame, w)
				return
			}
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
			match.tilesUnderAttack = []string{}
			getKing := match.pieces[kingName]
			getKingSquare := match.board[getKing.Tile]

			message = fmt.Sprintf(
				getSinglePieceMessage(),
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
				"",
			)

			err = sendMessage(onlineGame, found, w, message, [2][]int{
				{getKingSquare.CoordinatePosition[0]},
				{getKingSquare.CoordinatePosition[1]},
			})

			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
		cfg.endTurn(currentGame, w)
		return
	}

	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName && samePiece(match.selectedPiece, currentPiece) {

		isCastle, kingCheck := checkForCastle(match, currentPiece)

		if isCastle && !match.isBlackUnderCheck && !match.isWhiteUnderCheck && !kingCheck {

			err := cfg.handleCastle(w, currentPiece, currentGame, r)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error with handling castle", err)
			}
			return
		}

		var kingsName string
		var className string
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			kingsName = "white_king"
		} else if !match.isWhiteTurn && match.isBlackUnderCheck {
			kingsName = "black_king"
		}

		if kingsName != "" && strings.Contains(match.selectedPiece.Name, kingsName) {
			className = `class="bg-red-400"`
		}

		_, err := fmt.Fprintf(
			w,
			getReselectPieceMessage(),
			currentPieceName,
			currentSquare.CoordinatePosition[0]*multiplier,
			currentSquare.CoordinatePosition[1]*multiplier,
			currentPiece.Image,
			match.selectedPiece.Name,
			selSq.CoordinatePosition[0]*multiplier,
			selSq.CoordinatePosition[1]*multiplier,
			match.selectedPiece.Image,
			className,
		)

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't send to page", err)
		}

		match.selectedPiece = currentPiece
		cfg.Matches[currentGame] = match
		return
	}

	if currentSquare.Selected {
		currentSquare.Selected = false
		isKing := match.selectedPiece.IsKing
		match.selectedPiece = components.Piece{}
		match.board[currentSquareName] = currentSquare
		var kingsName string
		var className string
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			kingsName = "white_king"
		} else if !match.isWhiteTurn && match.isBlackUnderCheck {
			kingsName = "black_king"
		}
		if kingsName != "" && isKing {
			className = `class="bg-red-400"`
		}
		_, err := fmt.Fprintf(
			w,
			getSinglePieceMessage(),
			currentPieceName,
			currentSquare.CoordinatePosition[0]*multiplier,
			currentSquare.CoordinatePosition[1]*multiplier,
			currentPiece.Image,
			className,
		)

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		}

		cfg.Matches[currentGame] = match

		return
	} else {
		currentSquare.Selected = true
		match.selectedPiece = currentPiece
		match.board[currentSquareName] = currentSquare
		className := `class="bg-sky-300"`
		_, err := fmt.Fprintf(
			w,
			getSinglePieceMessage(),
			currentPieceName,
			currentSquare.CoordinatePosition[0]*multiplier,
			currentSquare.CoordinatePosition[1]*multiplier,
			currentPiece.Image,
			className,
		)

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
		cfg.Matches[currentGame] = match
		return
	}
}

func (cfg *appConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}
	currentGame := c.Value
	onlineGame, found := cfg.connections[currentGame]
	match := cfg.Matches[currentGame]
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile

	legalMoves := checkLegalMoves(match)

	var kingCheck bool
	if match.selectedPiece.IsKing && slices.Contains(legalMoves, currentSquareName) {
		kingCheck = handleChecksWhenKingMoves(currentSquareName, match)
	} else if !slices.Contains(legalMoves, currentSquareName) && !slices.Contains(legalMoves, fmt.Sprintf("enpessant_%v", currentSquareName)) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var check bool
	if !match.selectedPiece.IsKing {
		check, _, _ = cfg.handleCheckForCheck(currentSquareName, currentGame, match.selectedPiece)
	}

	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if slices.Contains(legalMoves, fmt.Sprintf("enpessant_%v", currentSquareName)) {
		var squareToDeleteName string
		var userColor string
		if strings.Contains(match.possibleEnPessant, "white") {
			enPessantSlice := strings.Split(match.possibleEnPessant, "_")
			squareNumber, _ := strconv.Atoi(string(enPessantSlice[1][0]))
			squareToDeleteName = fmt.Sprintf("%v%v", squareNumber-1, string(enPessantSlice[1][1]))
			userColor = "white"
		} else {
			enPessantSlice := strings.Split(match.possibleEnPessant, "_")
			squareNumber, _ := strconv.Atoi(string(enPessantSlice[1][0]))
			squareToDeleteName = fmt.Sprintf("%v%v", squareNumber+1, string(enPessantSlice[1][1]))
			userColor = "black"
		}
		squareToDelete := match.board[squareToDeleteName]
		pieceToDelete := squareToDelete.Piece
		currentSquare := match.board[currentSquareName]
		message := fmt.Sprintf(
			getEatPiecesMessage(),
			pieceToDelete.Name,
			pieceToDelete.Image,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
			userColor,
			pieceToDelete.Image,
		)

		err = sendMessage(onlineGame, found, w, message, [2][]int{
			{currentSquare.CoordinatePosition[0]},
			{currentSquare.CoordinatePosition[1]},
		})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't print to page", err)
			return
		}

		match, squareToDelete, saveSelected := eatCleanup(match, pieceToDelete, squareToDeleteName, currentSquareName)

		cfg.Matches[currentGame] = match
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}

		noCheck, err := handleIfCheck(w, r, cfg, saveSelected, currentGame)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "handle check error: ", err)
			return
		}
		if noCheck {
			var kingName string
			if match.isWhiteUnderCheck {
				kingName = "white_king"
			} else if match.isBlackUnderCheck {
				kingName = "black_king"
			} else {
				cfg.endTurn(currentGame, w)
				return
			}
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
			match.tilesUnderAttack = []string{}
			getKing := match.pieces[kingName]
			getKingSquare := match.board[getKing.Tile]

			message = fmt.Sprintf(
				getSinglePieceMessage(),
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
				"",
			)

			err = sendMessage(onlineGame, found, w, message, [2][]int{
				{getKingSquare.CoordinatePosition[0]},
				{getKingSquare.CoordinatePosition[1]},
			})

			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
		cfg.endTurn(currentGame, w)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		message := fmt.Sprintf(
			getSinglePieceMessage(),
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
			"",
		)

		err = sendMessage(onlineGame, found, w, message, [2][]int{
			{currentSquare.CoordinatePosition[0]},
			{currentSquare.CoordinatePosition[1]},
		})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
		match = checkForEnPessant(selectedSquare, currentSquare, match)
		saveSelected := match.selectedPiece
		match.allMoves = append(match.allMoves, currentSquareName)
		bigCleanup(currentSquareName, &match)
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}
		match.movesSinceLastCapture++
		cfg.Matches[currentGame] = match
		noCheck, err := handleIfCheck(w, r, cfg, saveSelected, currentGame)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		}
		if noCheck {
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
			cfg.Matches[currentGame] = match
		}
		pawnPromotion, err := cfg.checkForPawnPromotion(saveSelected.Name, currentGame, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error checking pawn promotion", err)
		}
		if saveSelected.IsPawn && pawnPromotion {
			return
		}
		cfg.endTurn(currentGame, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) coverCheckHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	currentGame := c.Value
	onlineGame, found := cfg.connections[currentGame]
	match := cfg.Matches[currentGame]
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile

	legalMoves := checkLegalMoves(match)

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var check bool
	var kingCheck bool
	if match.selectedPiece.IsKing {
		kingCheck = handleChecksWhenKingMoves(currentSquareName, match)
	} else {
		check, _, _ = cfg.handleCheckForCheck(currentSquareName, currentGame, match.selectedPiece)
	}
	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var kingName string

	if match.isWhiteTurn {
		kingName = "white_king"
	} else {
		kingName = "black_king"
	}

	king := match.pieces[kingName]
	kingSquare := match.board[king.Tile]

	if selectedSquare != "" && selectedSquare != currentSquareName {
		message := fmt.Sprintf(
			getCoverCheckMessage(),
			currentSquareName,
			currentSquare.Color,
			king.Name,
			kingSquare.Coordinates[0],
			kingSquare.Coordinates[1],
			king.Image,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
		)

		err = sendMessage(onlineGame, found, w, message, [2][]int{
			{
				kingSquare.CoordinatePosition[0],
				currentSquare.CoordinatePosition[0],
			},
			{
				currentSquare.CoordinatePosition[1],
				kingSquare.CoordinatePosition[1],
			},
		})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
		saveSelected := match.selectedPiece
		match.allMoves = append(match.allMoves, currentSquareName)
		bigCleanup(currentSquareName, &match)
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}

		for _, tile := range match.tilesUnderAttack {
			t := match.board[tile]
			if t.Piece.Name != "" {
				err := respondWithNewPiece(w, r, t)

				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, "error with new piece", err)
					return
				}
			} else {
				message := fmt.Sprintf(
					getTileMessage(),
					tile,
					"move-to",
					t.Color,
				)
				err = sendMessage(onlineGame, found, w, message, [2][]int{})
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, "Couldn't write to page", err)
					return
				}

			}
		}

		pawnPromotion, err := cfg.checkForPawnPromotion(saveSelected.Name, currentGame, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "check pawn promotion error", err)
		}
		if saveSelected.IsPawn && pawnPromotion {
			return
		}

		noCheck, err := handleIfCheck(w, r, cfg, saveSelected, currentGame)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "handle check error", err)
		}
		if noCheck {
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
		}

		match.possibleEnPessant = ""
		match.movesSinceLastCapture++
		cfg.Matches[currentGame] = match
		cfg.endTurn(currentGame, w)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) timerHandler(w http.ResponseWriter, r *http.Request) {

	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	} else if strings.Contains(c.Value, "database:") {
		return
	}
	currentGame := c.Value
	onlineGame, found := cfg.connections[currentGame]
	match := cfg.Matches[currentGame]

	var toChangeColor string
	var stayTheSameColor string
	var toChange int
	var stayTheSame int

	if match.isWhiteTurn {
		toChangeColor = "white"
		match.whiteTimer -= 1
		toChange = match.whiteTimer
		stayTheSame = match.blackTimer
		stayTheSameColor = "black"
	} else {
		match.blackTimer -= 1
		toChangeColor = "black"
		toChange = match.blackTimer
		stayTheSame = match.whiteTimer
		stayTheSameColor = "white"
	}

	message := fmt.Sprintf(
		getTimerMessage(),
		toChangeColor,
		formatTime(toChange),
		stayTheSameColor,
		formatTime(stayTheSame),
	)

	err = sendMessage(onlineGame, found, w, message, [2][]int{})

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}

	cfg.Matches[currentGame] = match

	if match.isWhiteTurn && (match.whiteTimer < 0 || match.whiteTimer == 0) {
		msg, err := TemplString(components.EndGameModal("0-1", "black"))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
			return
		}

		err = sendMessage(onlineGame, found, w, msg, [2][]int{})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	} else if !match.isWhiteTurn && (match.blackTimer < 0 || match.blackTimer == 0) {
		msg, err := TemplString(components.EndGameModal("1-0", "white"))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
			return
		}

		err = sendMessage(onlineGame, found, w, msg, [2][]int{})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}
}

func (cfg *appConfig) handlePromotion(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	currentGame := cfg.Matches[c.Value]
	onlineGame, found := cfg.connections[c.Value]
	pawnName := r.FormValue("pawn")
	pieceName := r.FormValue("piece")

	allPieces := MakePieces()

	pawnPiece := currentGame.pieces[pawnName]

	newPiece := components.Piece{
		Name:       pawnName,
		Image:      allPieces[pieceName].Image,
		Tile:       pawnPiece.Tile,
		IsWhite:    pawnPiece.IsWhite,
		LegalMoves: allPieces[pieceName].LegalMoves,
		MovesOnce:  allPieces[pieceName].MovesOnce,
		Moved:      true,
		IsKing:     false,
		IsPawn:     false,
	}

	delete(currentGame.pieces, pawnName)
	currentGame.pieces[pawnName] = newPiece
	currentSquare := currentGame.board[pawnPiece.Tile]
	currentSquare.Piece = newPiece
	currentGame.board[pawnPiece.Tile] = currentSquare

	cfg.Matches[c.Value] = currentGame

	message := fmt.Sprintf(
		getPromotionDoneMessage(),
		pawnName,
		currentSquare.Coordinates[0],
		currentSquare.Coordinates[1],
		currentSquare.Piece.Image,
	)

	err = sendMessage(onlineGame, found, w, message, [2][]int{
		{currentSquare.CoordinatePosition[0]},
		{currentSquare.CoordinatePosition[1]},
	})

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}

	userId, err := cfg.isUserLoggedIn(r)
	if err != nil && !strings.Contains(err.Error(), "named cookie not present") {
		logError("user not authorized", err)
	}

	if userId != uuid.Nil {
		go func(w http.ResponseWriter, r *http.Request) {
			boardState := make(map[string]string, 0)
			for k, v := range currentGame.pieces {
				boardState[k] = v.Tile
			}

			jsonBoard, err := json.Marshal(boardState)

			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error marshaling board state", err)
				return
			}

			moveDB, err := cfg.database.GetLatestMoveForMatch(r.Context(), currentGame.matchId)

			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "database erro", err)
				return
			}

			err = cfg.database.UpdateBoardForMove(r.Context(), database.UpdateBoardForMoveParams{
				Board:   jsonBoard,
				MatchID: moveDB.MatchID,
				Move:    moveDB.Move,
			})
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "Couldn't update board for move", err)
				return
			}
		}(w, r)
	}

	noCheck, err := handleIfCheck(w, r, cfg, newPiece, c.Value)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "error with handle check", err)
		return
	}
	if noCheck && (currentGame.isBlackUnderCheck || currentGame.isWhiteUnderCheck) {
		var kingName string
		if currentGame.isWhiteUnderCheck {
			kingName = "white_king"
		} else if currentGame.isBlackUnderCheck {
			kingName = "black_king"
		} else {
			cfg.endTurn(c.Value, w)
			return
		}

		currentGame.isWhiteUnderCheck = false
		currentGame.isBlackUnderCheck = false
		currentGame.tilesUnderAttack = []string{}
		getKing := currentGame.pieces[kingName]
		getKingSquare := currentGame.board[getKing.Tile]

		message := fmt.Sprintf(
			getSinglePieceMessage(),
			getKing.Name,
			getKingSquare.Coordinates[0],
			getKingSquare.Coordinates[1],
			getKing.Image,
			"",
		)

		err = sendMessage(onlineGame, found, w, message, [2][]int{
			{getKingSquare.CoordinatePosition[0]},
			{getKingSquare.CoordinatePosition[1]},
		})

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}

	currentGame.possibleEnPessant = ""
	currentGame.movesSinceLastCapture++
	cfg.Matches[c.Value] = currentGame
	cfg.endTurn(c.Value, w)
}

func (cfg *appConfig) endGameHandler(w http.ResponseWriter, r *http.Request) {
	currentGame, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	err = r.ParseForm()
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "error parsing form", err)
		return
	}
	saveGame := cfg.Matches[currentGame.Value]
	delete(cfg.Matches, currentGame.Value)
	err = cfg.database.UpdateMatchOnEnd(r.Context(), database.UpdateMatchOnEndParams{
		Result: r.FormValue("result"),
		ID:     saveGame.matchId,
	})
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "error updating match", err)
		return
	}
	if match, ok := cfg.connections[currentGame.Value]; ok {
		_ = match.players["white"].Conn.Close()
		_ = match.players["black"].Conn.Close()
		delete(cfg.connections, currentGame.Value)
	}
	cGC := http.Cookie{
		Name:     "current_game",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cGC)
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) surrenderHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	uC, err := r.Cookie("access_token")
	if err == nil && uC.Value != "" && strings.Contains(c.Value, "online:") {
		var msg string
		connection := cfg.connections[c.Value]
		userId, err := auth.ValidateJWT(uC.Value, cfg.secret)
		if err != nil {
			respondWithAnError(w, http.StatusUnauthorized, "user not found", err)
			return
		}
		if connection.players["white"].ID == userId {
			msg, err = TemplString(components.EndGameModal("0-1", "black"))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
				return
			}
		} else if connection.players["black"].ID == userId {
			msg, err = TemplString(components.EndGameModal("1-0", "white"))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
				return
			}
		}
		err = connection.players["white"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "writing online message error", err)
			return
		}
		err = connection.players["black"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "writing online message error", err)
			return
		}
		return
	}
	currentGame := cfg.Matches[c.Value]
	if currentGame.isWhiteTurn {
		err := components.EndGameModal("0-1", "black").Render(r.Context(), w)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error writing the end game modal", err)
			return
		}
	} else {
		err := components.EndGameModal("1-0", "white").Render(r.Context(), w)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error writing the end game modal", err)
			return
		}
	}
}

func (cfg *appConfig) handleCastle(w http.ResponseWriter, currentPiece components.Piece, currentGame string, r *http.Request) error {
	match := cfg.Matches[currentGame]
	onlineGame, found := cfg.connections[currentGame]

	var king components.Piece
	var rook components.Piece
	var multiplier int

	if match.selectedPiece.IsKing {
		king = match.selectedPiece
		rook = currentPiece
	} else {
		king = currentPiece
		rook = match.selectedPiece
	}

	if found {
		userC, err := r.Cookie("access_token")

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
			return err
		}

		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
			return err
		}

		for _, player := range onlineGame.players {
			if player.ID == userId {
				multiplier = player.Multiplier
			}
		}
	} else {
		multiplier = match.coordinateMultiplier
	}

	kTile := king.Tile
	rTile := rook.Tile
	savedKingTile := match.board[king.Tile]
	savedRookTile := match.board[rook.Tile]
	kingSquare := match.board[king.Tile]
	rookSquare := match.board[rook.Tile]

	if kingSquare.Coordinates[1] < rookSquare.Coordinates[1] {
		kC := kingSquare.Coordinates[1]
		rookSquare.Coordinates[1] = kC + multiplier
		kingSquare.Coordinates[1] = kC + multiplier*2
		rookSquare.CoordinatePosition[1] = rookSquare.Coordinates[1] / multiplier
		kingSquare.CoordinatePosition[1] = kingSquare.Coordinates[1] / multiplier
	} else {
		kC := kingSquare.Coordinates[1]
		rookSquare.Coordinates[1] = kC - multiplier
		kingSquare.Coordinates[1] = kC - multiplier*2
		rookSquare.CoordinatePosition[1] = rookSquare.Coordinates[1] / multiplier
		kingSquare.CoordinatePosition[1] = kingSquare.Coordinates[1] / multiplier
	}

	message := fmt.Sprintf(
		getCastleMessage(),
		king.Name,
		kingSquare.Coordinates[0],
		kingSquare.Coordinates[1],
		king.Image,
		rook.Name,
		rookSquare.Coordinates[0],
		rookSquare.Coordinates[1],
		rook.Image,
	)

	err := sendMessage(onlineGame, found, w, message, [2][]int{
		{
			kingSquare.CoordinatePosition[0],
			rookSquare.CoordinatePosition[0],
		},
		{
			kingSquare.CoordinatePosition[1],
			rookSquare.CoordinatePosition[1],
		},
	})

	if err != nil {
		return err
	}

	rowIdx := rowIdxMap[string(king.Tile[0])]
	king.Tile = mockBoard[rowIdx][kingSquare.Coordinates[1]/multiplier]
	rook.Tile = mockBoard[rowIdx][rookSquare.Coordinates[1]/multiplier]
	king.Moved = true
	rook.Moved = true
	newKingSquare := match.board[king.Tile]
	newRookSquare := match.board[rook.Tile]
	newKingSquare.Piece = king
	newRookSquare.Piece = rook
	match.board[king.Tile] = newKingSquare
	match.board[rook.Tile] = newRookSquare
	match.pieces[king.Name] = king
	match.pieces[rook.Name] = rook
	savedKingTile.Piece = components.Piece{}
	savedRookTile.Piece = components.Piece{}
	match.board[kTile] = savedKingTile
	match.board[rTile] = savedRookTile
	match.selectedPiece = components.Piece{}
	match.isWhiteTurn = !match.isWhiteTurn
	match.possibleEnPessant = ""
	match.movesSinceLastCapture++
	cfg.Matches[currentGame] = match

	if kingSquare.CoordinatePosition[1]-rookSquare.CoordinatePosition[1] == 1 {
		match.allMoves = append(match.allMoves, "O-O")
		err := cfg.showMoves(match, "O-O", "king", w, r)
		if err != nil {
			return err
		}
	} else {
		match.allMoves = append(match.allMoves, "O-O-O")
		err := cfg.showMoves(match, "O-O-O", "king", w, r)
		if err != nil {
			return err
		}
	}

	cfg.gameDone(match, currentGame, w)

	return nil
}
