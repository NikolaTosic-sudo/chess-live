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
	"github.com/NikolaTosic-sudo/chess-live/internal/matches"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/NikolaTosic-sudo/chess-live/internal/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (cfg *appConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentPieceName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}
	err = r.ParseForm()

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	multiplier, err := strconv.Atoi(r.FormValue("multiplier"))

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return
	}
	currentGame := c.Value
	match, _ := cfg.Matches.GetMatch(currentGame)
	onlineGame, found := match.IsOnlineMatch()
	currentPiece := match.Pieces[currentPieceName]
	userC, err := r.Cookie("access_token")

	var userId uuid.UUID
	if err == nil && userC.Value != "" {
		userId, _ = auth.ValidateJWT(userC.Value, cfg.secret)
	}

	canPlay := match.CanPlay(currentPiece, onlineGame.Players, userId)

	currentSquareName := currentPiece.Tile
	currentSquare := match.Board[currentSquareName]
	selectedSquare := match.SelectedPiece.Tile
	selSq := match.Board[selectedSquare]
	legalMoves := match.CheckLegalMoves()

	if matches.CanEat(match.SelectedPiece, currentPiece) && slices.Contains(legalMoves, currentSquareName) {
		if found {
			if match.IsWhiteTurn && onlineGame.Players["white"].ID != userId {
				return
			} else if !match.IsWhiteTurn && onlineGame.Players["black"].ID != userId {
				return
			}
		}
		var kingCheck bool
		if match.SelectedPiece.IsKing {
			kingCheck = match.HandleChecksWhenKingMoves(currentSquareName)
		} else if match.IsWhiteTurn && match.IsWhiteUnderCheck && !slices.Contains(match.TilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		} else if !match.IsWhiteTurn && match.IsBlackUnderCheck && !slices.Contains(match.TilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var check bool
		if !match.SelectedPiece.IsKing {
			check, _, _ = match.HandleCheckForCheck(currentSquareName, match.SelectedPiece)
		}

		if check || kingCheck {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var userColor string
		if match.IsWhiteTurn {
			match.TakenPiecesWhite = append(match.TakenPiecesWhite, currentPiece.Image)
			userColor = "white"
		} else {
			match.TakenPiecesBlack = append(match.TakenPiecesBlack, currentPiece.Image)
			userColor = "black"
		}

		message := fmt.Sprintf(
			responses.GetEatPiecesMessage(),
			currentPiece.Name,
			currentPiece.Image,
			match.SelectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.SelectedPiece.Image,
			userColor,
			currentPiece.Image,
		)

		err = match.SendMessage(w, message, [2][]int{
			{currentSquare.CoordinatePosition[0]},
			{currentSquare.CoordinatePosition[1]},
		})

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't print to page", err)
			return
		}

		_, saveSelected := match.EatCleanup(currentPiece, selectedSquare, currentSquareName)

		cfg.Matches.SetMatch(currentGame, match)
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}
		pawnPromotion, err := match.CheckForPawnPromotion(saveSelected.Name, w, userId)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "pawn promotion error: ", err)
			return
		}

		if saveSelected.IsPawn && pawnPromotion {
			return
		}

		noCheck, err := match.HandleIfCheck(w, r, saveSelected)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "handle check error: ", err)
			return
		}
		if noCheck {
			var kingName string
			if match.IsWhiteUnderCheck {
				kingName = "white_king"
			} else if match.IsBlackUnderCheck {
				kingName = "black_king"
			} else {
				match.EndTurn(w)
				cfg.Matches.SetMatch(currentGame, match)
				return
			}
			match.IsWhiteUnderCheck = false
			match.IsBlackUnderCheck = false
			match.TilesUnderAttack = []string{}
			getKing := match.Pieces[kingName]
			getKingSquare := match.Board[getKing.Tile]

			message = fmt.Sprintf(
				responses.GetSinglePieceMessage(),
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
				"",
			)

			err = match.SendMessage(w, message, [2][]int{
				{getKingSquare.CoordinatePosition[0]},
				{getKingSquare.CoordinatePosition[1]},
			})

			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
		match.EndTurn(w)
		cfg.Matches.SetMatch(currentGame, match)
		return
	}

	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName && matches.SamePiece(match.SelectedPiece, currentPiece) {

		isCastle, kingCheck := match.CheckForCastle(currentPiece)

		if isCastle && !match.IsBlackUnderCheck && !match.IsWhiteUnderCheck && !kingCheck {

			err := cfg.handleCastle(w, currentPiece, currentGame, r)
			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "error with handling castle", err)
			}
			return
		}

		var kingsName string
		var className string
		if match.IsWhiteTurn && match.IsWhiteUnderCheck {
			kingsName = "white_king"
		} else if !match.IsWhiteTurn && match.IsBlackUnderCheck {
			kingsName = "black_king"
		}

		if kingsName != "" && strings.Contains(match.SelectedPiece.Name, kingsName) {
			className = `class="bg-red-400"`
		}

		_, err := fmt.Fprintf(
			w,
			responses.GetReselectPieceMessage(),
			currentPieceName,
			currentSquare.CoordinatePosition[0]*multiplier,
			currentSquare.CoordinatePosition[1]*multiplier,
			currentPiece.Image,
			match.SelectedPiece.Name,
			selSq.CoordinatePosition[0]*multiplier,
			selSq.CoordinatePosition[1]*multiplier,
			match.SelectedPiece.Image,
			className,
		)

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't send to page", err)
		}

		match.SelectedPiece = currentPiece
		cfg.Matches.SetMatch(currentGame, match)
		return
	}

	if currentSquare.Selected {
		currentSquare.Selected = false
		isKing := match.SelectedPiece.IsKing
		match.SelectedPiece = components.Piece{}
		match.Board[currentSquareName] = currentSquare
		var kingsName string
		var className string
		if match.IsWhiteTurn && match.IsWhiteUnderCheck {
			kingsName = "white_king"
		} else if !match.IsWhiteTurn && match.IsBlackUnderCheck {
			kingsName = "black_king"
		}
		if kingsName != "" && isKing {
			className = `class="bg-red-400"`
		}
		_, err := fmt.Fprintf(
			w,
			responses.GetSinglePieceMessage(),
			currentPieceName,
			currentSquare.CoordinatePosition[0]*multiplier,
			currentSquare.CoordinatePosition[1]*multiplier,
			currentPiece.Image,
			className,
		)

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		}

		cfg.Matches.SetMatch(currentGame, match)

		return
	} else {
		currentSquare.Selected = true
		match.SelectedPiece = currentPiece
		match.Board[currentSquareName] = currentSquare
		className := `class="bg-sky-300"`
		_, err := fmt.Fprintf(
			w,
			responses.GetSinglePieceMessage(),
			currentPieceName,
			currentSquare.CoordinatePosition[0]*multiplier,
			currentSquare.CoordinatePosition[1]*multiplier,
			currentPiece.Image,
			className,
		)

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
		cfg.Matches.SetMatch(currentGame, match)
		return
	}
}

func (cfg *appConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}
	currentGame := c.Value
	match, _ := cfg.Matches.GetMatch(currentGame)
	currentSquare := match.Board[currentSquareName]
	selectedSquare := match.SelectedPiece.Tile

	legalMoves := match.CheckLegalMoves()

	userId, _ := cfg.getUserId(r)

	var kingCheck bool
	if match.SelectedPiece.IsKing && slices.Contains(legalMoves, currentSquareName) {
		kingCheck = match.HandleChecksWhenKingMoves(currentSquareName)
	} else if !slices.Contains(legalMoves, currentSquareName) && !slices.Contains(legalMoves, fmt.Sprintf("enpessant_%v", currentSquareName)) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var check bool
	if !match.SelectedPiece.IsKing {
		check, _, _ = match.HandleCheckForCheck(currentSquareName, match.SelectedPiece)
	}

	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if slices.Contains(legalMoves, fmt.Sprintf("enpessant_%v", currentSquareName)) {
		var squareToDeleteName string
		var userColor string
		if strings.Contains(match.PossibleEnPessant, "white") {
			enPessantSlice := strings.Split(match.PossibleEnPessant, "_")
			squareNumber, _ := strconv.Atoi(string(enPessantSlice[1][0]))
			squareToDeleteName = fmt.Sprintf("%v%v", squareNumber-1, string(enPessantSlice[1][1]))
			userColor = "white"
		} else {
			enPessantSlice := strings.Split(match.PossibleEnPessant, "_")
			squareNumber, _ := strconv.Atoi(string(enPessantSlice[1][0]))
			squareToDeleteName = fmt.Sprintf("%v%v", squareNumber+1, string(enPessantSlice[1][1]))
			userColor = "black"
		}
		squareToDelete := match.Board[squareToDeleteName]
		pieceToDelete := squareToDelete.Piece
		currentSquare := match.Board[currentSquareName]
		message := fmt.Sprintf(
			responses.GetEatPiecesMessage(),
			pieceToDelete.Name,
			pieceToDelete.Image,
			match.SelectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.SelectedPiece.Image,
			userColor,
			pieceToDelete.Image,
		)

		err = match.SendMessage(w, message, [2][]int{
			{currentSquare.CoordinatePosition[0]},
			{currentSquare.CoordinatePosition[1]},
		})

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't print to page", err)
			return
		}

		squareToDelete, saveSelected := match.EatCleanup(pieceToDelete, squareToDeleteName, currentSquareName)

		cfg.Matches.SetMatch(currentGame, match)
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}

		noCheck, err := match.HandleIfCheck(w, r, saveSelected)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "handle check error: ", err)
			return
		}
		if noCheck {
			var kingName string
			if match.IsWhiteUnderCheck {
				kingName = "white_king"
			} else if match.IsBlackUnderCheck {
				kingName = "black_king"
			} else {
				match.EndTurn(w)
				cfg.Matches.SetMatch(currentGame, match)
				return
			}
			match.IsWhiteUnderCheck = false
			match.IsBlackUnderCheck = false
			match.TilesUnderAttack = []string{}
			getKing := match.Pieces[kingName]
			getKingSquare := match.Board[getKing.Tile]

			message = fmt.Sprintf(
				responses.GetSinglePieceMessage(),
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
				"",
			)

			err = match.SendMessage(w, message, [2][]int{
				{getKingSquare.CoordinatePosition[0]},
				{getKingSquare.CoordinatePosition[1]},
			})

			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}

		match.EndTurn(w)
		cfg.Matches.SetMatch(currentGame, match)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		message := fmt.Sprintf(
			responses.GetSinglePieceMessage(),
			match.SelectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.SelectedPiece.Image,
			"",
		)

		err = match.SendMessage(w, message, [2][]int{
			{currentSquare.CoordinatePosition[0]},
			{currentSquare.CoordinatePosition[1]},
		})

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
		match.CheckForEnPessant(selectedSquare, currentSquare)
		saveSelected := match.SelectedPiece
		match.AllMoves = append(match.AllMoves, currentSquareName)
		match.BigCleanup(currentSquareName)
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}
		match.MovesSinceLastCapture++
		cfg.Matches.SetMatch(currentGame, match)
		noCheck, err := match.HandleIfCheck(w, r, saveSelected)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		}
		if noCheck {
			match.IsWhiteUnderCheck = false
			match.IsBlackUnderCheck = false
			cfg.Matches.SetMatch(currentGame, match)
		}
		pawnPromotion, err := match.CheckForPawnPromotion(saveSelected.Name, w, userId)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "error checking pawn promotion", err)
		}
		if saveSelected.IsPawn && pawnPromotion {
			return
		}
		match.EndTurn(w)
		cfg.Matches.SetMatch(currentGame, match)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) coverCheckHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	currentGame := c.Value
	match, _ := cfg.Matches.GetMatch(currentGame)
	currentSquare := match.Board[currentSquareName]
	selectedSquare := match.SelectedPiece.Tile

	legalMoves := match.CheckLegalMoves()

	userId, _ := cfg.getUserId(r)

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var check bool
	var kingCheck bool
	if match.SelectedPiece.IsKing {
		kingCheck = match.HandleChecksWhenKingMoves(currentSquareName)
	} else {
		check, _, _ = match.HandleCheckForCheck(currentSquareName, match.SelectedPiece)
	}
	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var kingName string

	if match.IsWhiteTurn {
		kingName = "white_king"
	} else {
		kingName = "black_king"
	}

	king := match.Pieces[kingName]
	kingSquare := match.Board[king.Tile]

	if selectedSquare != "" && selectedSquare != currentSquareName {
		message := fmt.Sprintf(
			responses.GetCoverCheckMessage(),
			currentSquareName,
			currentSquare.Color,
			king.Name,
			kingSquare.Coordinates[0],
			kingSquare.Coordinates[1],
			king.Image,
			match.SelectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.SelectedPiece.Image,
		)

		err = match.SendMessage(w, message, [2][]int{
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
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
		saveSelected := match.SelectedPiece
		match.AllMoves = append(match.AllMoves, currentSquareName)
		match.BigCleanup(currentSquareName)
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}

		for _, tile := range match.TilesUnderAttack {
			t := match.Board[tile]
			if t.Piece.Name != "" {
				err := responses.RespondWithNewPiece(w, r, t)

				if err != nil {
					responses.RespondWithAnError(w, http.StatusInternalServerError, "error with new piece", err)
					return
				}
			} else {
				message := fmt.Sprintf(
					responses.GetTileMessage(),
					tile,
					"move-to",
					t.Color,
				)
				err = match.SendMessage(w, message, [2][]int{})
				if err != nil {
					responses.RespondWithAnError(w, http.StatusInternalServerError, "Couldn't write to page", err)
					return
				}

			}
		}

		pawnPromotion, err := match.CheckForPawnPromotion(saveSelected.Name, w, userId)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "check pawn promotion error", err)
		}
		if saveSelected.IsPawn && pawnPromotion {
			return
		}

		noCheck, err := match.HandleIfCheck(w, r, saveSelected)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "handle check error", err)
		}
		if noCheck {
			match.IsWhiteUnderCheck = false
			match.IsBlackUnderCheck = false
		}

		match.PossibleEnPessant = ""
		match.MovesSinceLastCapture++
		cfg.Matches.SetMatch(currentGame, match)
		match.EndTurn(w)
		cfg.Matches.SetMatch(currentGame, match)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) timerHandler(w http.ResponseWriter, r *http.Request) {

	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	} else if strings.Contains(c.Value, "database:") {
		return
	}
	currentGame := c.Value
	match, _ := cfg.Matches.GetMatch(currentGame)

	var toChangeColor string
	var stayTheSameColor string
	var toChange int
	var stayTheSame int

	if match.IsWhiteTurn {
		toChangeColor = "white"
		match.WhiteTimer -= 1
		toChange = match.WhiteTimer
		stayTheSame = match.BlackTimer
		stayTheSameColor = "black"
	} else {
		match.BlackTimer -= 1
		toChangeColor = "black"
		toChange = match.BlackTimer
		stayTheSame = match.WhiteTimer
		stayTheSameColor = "white"
	}

	message := fmt.Sprintf(
		responses.GetTimerMessage(),
		toChangeColor,
		utils.FormatTime(toChange),
		stayTheSameColor,
		utils.FormatTime(stayTheSame),
	)

	err = match.SendMessage(w, message, [2][]int{})

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}

	cfg.Matches.SetMatch(currentGame, match)

	if match.IsWhiteTurn && (match.WhiteTimer < 0 || match.WhiteTimer == 0) {
		msg, err := utils.TemplString(components.EndGameModal("0-1", "black"))
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
			return
		}

		err = match.SendMessage(w, msg, [2][]int{})

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	} else if !match.IsWhiteTurn && (match.BlackTimer < 0 || match.BlackTimer == 0) {
		msg, err := utils.TemplString(components.EndGameModal("1-0", "white"))
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
			return
		}

		err = match.SendMessage(w, msg, [2][]int{})

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}
}

func (cfg *appConfig) handlePromotion(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	currentGameName := c.Value
	currentGame, _ := cfg.Matches.GetMatch(currentGameName)
	pawnName := r.FormValue("pawn")
	pieceName := r.FormValue("piece")

	allPieces := MakePieces()

	pawnPiece := currentGame.Pieces[pawnName]

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

	delete(currentGame.Pieces, pawnName)
	currentGame.Pieces[pawnName] = newPiece
	currentSquare := currentGame.Board[pawnPiece.Tile]
	currentSquare.Piece = newPiece
	currentGame.Board[pawnPiece.Tile] = currentSquare

	cfg.Matches.SetMatch(c.Value, currentGame)

	message := fmt.Sprintf(
		responses.GetPromotionDoneMessage(),
		pawnName,
		currentSquare.Coordinates[0],
		currentSquare.Coordinates[1],
		currentSquare.Piece.Image,
	)

	err = currentGame.SendMessage(w, message, [2][]int{
		{currentSquare.CoordinatePosition[0]},
		{currentSquare.CoordinatePosition[1]},
	})

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}

	userId, err := cfg.isUserLoggedIn(r)
	if err != nil && !strings.Contains(err.Error(), "named cookie not present") {
		responses.LogError("user not authorized", err)
	}

	if userId != uuid.Nil {
		go func(w http.ResponseWriter, r *http.Request) {
			boardState := make(map[string]string, 0)
			for k, v := range currentGame.Pieces {
				boardState[k] = v.Tile
			}

			jsonBoard, err := json.Marshal(boardState)

			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "error marshaling board state", err)
				return
			}

			moveDB, err := cfg.database.GetLatestMoveForMatch(r.Context(), currentGame.MatchId)

			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "database erro", err)
				return
			}

			err = cfg.database.UpdateBoardForMove(r.Context(), database.UpdateBoardForMoveParams{
				Board:   jsonBoard,
				MatchID: moveDB.MatchID,
				Move:    moveDB.Move,
			})
			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "Couldn't update board for move", err)
				return
			}
		}(w, r)
	}

	noCheck, err := currentGame.HandleIfCheck(w, r, newPiece)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error with handle check", err)
		return
	}
	if noCheck && (currentGame.IsBlackUnderCheck || currentGame.IsWhiteUnderCheck) {
		var kingName string
		if currentGame.IsWhiteUnderCheck {
			kingName = "white_king"
		} else if currentGame.IsBlackUnderCheck {
			kingName = "black_king"
		} else {
			currentGame.EndTurn(w)
			cfg.Matches.SetMatch(currentGameName, currentGame)
			return
		}

		currentGame.IsWhiteUnderCheck = false
		currentGame.IsBlackUnderCheck = false
		currentGame.TilesUnderAttack = []string{}
		getKing := currentGame.Pieces[kingName]
		getKingSquare := currentGame.Board[getKing.Tile]

		message := fmt.Sprintf(
			responses.GetSinglePieceMessage(),
			getKing.Name,
			getKingSquare.Coordinates[0],
			getKingSquare.Coordinates[1],
			getKing.Image,
			"",
		)

		err = currentGame.SendMessage(w, message, [2][]int{
			{getKingSquare.CoordinatePosition[0]},
			{getKingSquare.CoordinatePosition[1]},
		})

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}

	currentGame.PossibleEnPessant = ""
	currentGame.MovesSinceLastCapture++
	cfg.Matches.SetMatch(currentGameName, currentGame)
	currentGame.EndTurn(w)
	cfg.Matches.SetMatch(currentGameName, currentGame)
}

func (cfg *appConfig) endGameHandler(w http.ResponseWriter, r *http.Request) {
	currentGame, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}

	err = r.ParseForm()
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error parsing form", err)
		return
	}

	saveGame, _ := cfg.Matches.GetMatch(currentGame.Value)
	if match, ok := saveGame.IsOnlineMatch(); ok {
		_ = match.Players["white"].Conn.Close()
		_ = match.Players["black"].Conn.Close()
	}

	delete(cfg.Matches.Matches, currentGame.Value)

	err = cfg.database.UpdateMatchOnEnd(r.Context(), database.UpdateMatchOnEndParams{
		Result: r.FormValue("result"),
		ID:     saveGame.MatchId,
	})
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error updating match", err)
		return
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
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	uC, err := r.Cookie("access_token")
	currentGame, _ := cfg.Matches.GetMatch(c.Value)

	if err == nil && uC.Value != "" && strings.Contains(c.Value, "online:") {
		var msg string
		connection := currentGame.Online
		userId, err := auth.ValidateJWT(uC.Value, cfg.secret)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusUnauthorized, "user not found", err)
			return
		}
		if connection.Players["white"].ID == userId {
			msg, err = utils.TemplString(components.EndGameModal("0-1", "black"))
			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
				return
			}
		} else if connection.Players["black"].ID == userId {
			msg, err = utils.TemplString(components.EndGameModal("1-0", "white"))
			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
				return
			}
		}
		err = connection.Players["white"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "writing online message error", err)
			return
		}
		err = connection.Players["black"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "writing online message error", err)
			return
		}
		return
	}
	if currentGame.IsWhiteTurn {
		err := components.EndGameModal("0-1", "black").Render(r.Context(), w)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "error writing the end game modal", err)
			return
		}
	} else {
		err := components.EndGameModal("1-0", "white").Render(r.Context(), w)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "error writing the end game modal", err)
			return
		}
	}
}

func (cfg *appConfig) handleCastle(w http.ResponseWriter, currentPiece components.Piece, currentGame string, r *http.Request) error {
	match, _ := cfg.Matches.GetMatch(currentGame)
	onlineGame, found := match.IsOnlineMatch()

	var king components.Piece
	var rook components.Piece
	var multiplier int

	if match.SelectedPiece.IsKing {
		king = match.SelectedPiece
		rook = currentPiece
	} else {
		king = currentPiece
		rook = match.SelectedPiece
	}

	if found {
		userC, err := r.Cookie("access_token")

		if err != nil {
			responses.RespondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
			return err
		}

		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			responses.RespondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
			return err
		}

		for _, player := range onlineGame.Players {
			if player.ID == userId {
				multiplier = player.Multiplier
			}
		}
	} else {
		multiplier = match.CoordinateMultiplier
	}

	kTile := king.Tile
	rTile := rook.Tile
	savedKingTile := match.Board[king.Tile]
	savedRookTile := match.Board[rook.Tile]
	kingSquare := match.Board[king.Tile]
	rookSquare := match.Board[rook.Tile]

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
		responses.GetCastleMessage(),
		king.Name,
		kingSquare.Coordinates[0],
		kingSquare.Coordinates[1],
		king.Image,
		rook.Name,
		rookSquare.Coordinates[0],
		rookSquare.Coordinates[1],
		rook.Image,
	)

	err := match.SendMessage(w, message, [2][]int{
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

	rowIdx := matches.RowIdxMap[string(king.Tile[0])]
	king.Tile = matches.MockBoard[rowIdx][kingSquare.Coordinates[1]/multiplier]
	rook.Tile = matches.MockBoard[rowIdx][rookSquare.Coordinates[1]/multiplier]
	king.Moved = true
	rook.Moved = true
	newKingSquare := match.Board[king.Tile]
	newRookSquare := match.Board[rook.Tile]
	newKingSquare.Piece = king
	newRookSquare.Piece = rook
	match.Board[king.Tile] = newKingSquare
	match.Board[rook.Tile] = newRookSquare
	match.Pieces[king.Name] = king
	match.Pieces[rook.Name] = rook
	savedKingTile.Piece = components.Piece{}
	savedRookTile.Piece = components.Piece{}
	match.Board[kTile] = savedKingTile
	match.Board[rTile] = savedRookTile
	match.SelectedPiece = components.Piece{}
	match.IsWhiteTurn = !match.IsWhiteTurn
	match.PossibleEnPessant = ""
	match.MovesSinceLastCapture++
	cfg.Matches.SetMatch(currentGame, match)

	if kingSquare.CoordinatePosition[1]-rookSquare.CoordinatePosition[1] == 1 {
		match.AllMoves = append(match.AllMoves, "O-O")
		err := cfg.showMoves(match, "O-O", "king", w, r)
		if err != nil {
			return err
		}
	} else {
		match.AllMoves = append(match.AllMoves, "O-O-O")
		err := cfg.showMoves(match, "O-O-O", "king", w, r)
		if err != nil {
			return err
		}
	}

	match.GameDone(w)

	return nil
}
