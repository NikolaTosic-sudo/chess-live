package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	currentGame := c.Value
	onlineGame, found := cfg.connections[currentGame]
	match := cfg.Matches[currentGame]
	currentPiece := match.pieces[currentPieceName]
	canPlay, err := cfg.canPlay(currentPiece, currentGame, onlineGame, r)
	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
		return
	}
	currentSquareName := currentPiece.Tile
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile
	selSq := match.board[selectedSquare]
	legalMoves := cfg.checkLegalMoves(currentGame, Match{})

	if canEat(match.selectedPiece, currentPiece) && slices.Contains(legalMoves, currentSquareName) {
		if onlineGame != nil {
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

			if match.isWhiteTurn && onlineGame["white"].ID != userId {
				return
			} else if !match.isWhiteTurn && onlineGame["black"].ID != userId {
				return
			}
		}
		var kingCheck bool
		if match.selectedPiece.IsKing {
			kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName, currentGame, Match{})
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

		message := fmt.Sprintf(`
			<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="display: none">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<div id="lost-pieces-%v" hx-swap-oob="afterbegin">
				<img src="/assets/pieces/%v.svg" class="w-[18px] h-[18px]" />
			</div>
		`,
			currentPiece.Name,
			currentPiece.Image,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
			userColor,
			currentPiece.Image,
		)
		if found {
			for playerColor, onlinePlayer := range onlineGame {
				err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
					return
				}
			}
		} else {
			_, err := fmt.Fprint(w, message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't print to page", err)
				return
			}
		}
		match.allMoves = append(match.allMoves, currentSquareName)
		delete(match.pieces, currentPieceName)
		match.selectedPiece.Tile = currentSquareName
		match.selectedPiece.Moved = true
		match.pieces[match.selectedPiece.Name] = match.selectedPiece
		currentSquare.Piece = match.selectedPiece
		selSq.Piece = components.Piece{}
		match.board[currentSquareName] = currentSquare
		match.board[selectedSquare] = selSq
		saveSelected := match.selectedPiece
		match.selectedPiece = components.Piece{}
		match.possibleEnPessant = ""
		match.movesSinceLastCapture = 0

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

		noCheck, err := handleIfCheck(w, cfg, saveSelected, currentGame)
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
				cfg.endTurn(currentGame, r, w)
				return
			}
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
			match.tilesUnderAttack = []string{}
			getKing := match.pieces[kingName]
			getKingSquare := match.board[getKing.Tile]

			message = fmt.Sprintf(`
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
			)
			if found {
				for playerColor, onlinePlayer := range onlineGame {
					err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
					if err != nil {
						respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
						return
					}
				}
			} else {
				_, err := fmt.Fprint(w, message)
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
					return
				}
			}
		}
		cfg.endTurn(currentGame, r, w)
		return
	}

	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName && samePiece(match.selectedPiece, currentPiece) {

		isCastle, kingCheck := cfg.checkForCastle(match.board, match.selectedPiece, currentPiece, currentGame)

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

		_, err := fmt.Fprintf(w, `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
					<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
				</span>
	
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
					<img src="/assets/pieces/%v.svg" %v  />
				</span>
			`,
			currentPieceName,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			currentPiece.Image,
			match.selectedPiece.Name,
			selSq.Coordinates[0],
			selSq.Coordinates[1],
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
		_, err := fmt.Fprintf(w, `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
			currentPieceName,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
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
		_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300 " />
			</span>
		`, currentPieceName, currentSquare.Coordinates[0], currentSquare.Coordinates[1], currentPiece.Image)

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

	legalMoves := cfg.checkLegalMoves(currentGame, Match{})

	var kingCheck bool
	if match.selectedPiece.IsKing && slices.Contains(legalMoves, currentSquareName) {
		kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName, currentGame, Match{})
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
		message := fmt.Sprintf(`
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 1660c61 (feature: finally implemented en)
			<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="display: none">
				<img src="/assets/pieces/%v.svg" />
			</span>

<<<<<<< HEAD
=======
>>>>>>> b549cb9 (feature: En Pessant implementation WIP)
=======
>>>>>>> 1660c61 (feature: finally implemented en)
			<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<div id="lost-pieces-%v" hx-swap-oob="afterbegin">
				<img src="/assets/pieces/%v.svg" class="w-[18px] h-[18px]" />
			</div>
		`,
			pieceToDelete.Name,
			pieceToDelete.Image,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
			userColor,
			pieceToDelete.Image,
		)
		if found {
			for playerColor, onlinePlayer := range onlineGame {
				err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
					return
				}
			}
		} else {
			_, err := fmt.Fprint(w, message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't print to page", err)
				return
			}
		}

		match.allMoves = append(match.allMoves, currentSquareName)
		delete(match.pieces, pieceToDelete.Name)
		match.selectedPiece.Tile = currentSquareName
		match.pieces[match.selectedPiece.Name] = match.selectedPiece
		currentSquare.Piece = match.selectedPiece
		squareToDelete.Piece = components.Piece{}
		match.board[currentSquareName] = currentSquare
		match.board[squareToDeleteName] = squareToDelete
		saveSelected := match.selectedPiece
		match.selectedPiece = components.Piece{}
		match.possibleEnPessant = ""
		match.movesSinceLastCapture = 0
		cfg.Matches[currentGame] = match
		err = cfg.showMoves(match, currentSquareName, saveSelected.Name, w, r)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "show moves error: ", err)
			return
		}

		noCheck, err := handleIfCheck(w, cfg, saveSelected, currentGame)
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
				cfg.endTurn(currentGame, r, w)
				return
			}
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
			match.tilesUnderAttack = []string{}
			getKing := match.pieces[kingName]
			getKingSquare := match.board[getKing.Tile]

			message = fmt.Sprintf(`
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
			)
			if found {
				for playerColor, onlinePlayer := range onlineGame {
					err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
					if err != nil {
						respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
						return
					}
				}
			} else {
				_, err := fmt.Fprint(w, message)
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
					return
				}
			}
		}
		cfg.endTurn(currentGame, r, w)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		message := fmt.Sprintf(`
			<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
		)
		if found {
			for playerColor, onlinePlayer := range onlineGame {
				err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
				}
			}
		} else {
			_, err := fmt.Fprint(w, message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
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
		noCheck, err := handleIfCheck(w, cfg, saveSelected, currentGame)
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
		cfg.endTurn(currentGame, r, w)
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

	legalMoves := cfg.checkLegalMoves(currentGame, Match{})

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var check bool
	var kingCheck bool
	if match.selectedPiece.IsKing {
		kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName, currentGame, Match{})
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
		message := fmt.Sprintf(`
			<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="tile tile-md h-full w-full" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
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
		if found {
			for playerColor, onlinePlayer := range onlineGame {
				err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
					return
				}
			}
		} else {
			_, err := fmt.Fprint(w, message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
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
				err := respondWithNewPiece(w, t)

				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, "error with new piece", err)
					return
				}
			} else {
				message := fmt.Sprintf(`
						<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="tile tile-md" style="background-color: %v"></div>
				`,
					tile,
					t.Color,
				)
				if found {
					for playerColor, onlinePlayer := range onlineGame {
						err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
						if err != nil {
							respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
						}
					}
				} else {
					_, err := fmt.Fprint(w, message)
					if err != nil {
						respondWithAnError(w, http.StatusInternalServerError, "Couldn't write to page", err)
						return
					}
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

		noCheck, err := handleIfCheck(w, cfg, saveSelected, currentGame)
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
		cfg.endTurn(currentGame, r, w)

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

	message := fmt.Sprintf(`
	<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-white">%v</div>
	
	<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>
	
	`, toChangeColor, formatTime(toChange), stayTheSameColor, formatTime(stayTheSame))

	if found {
		for playerColor, onlinePlayer := range onlineGame {
			err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				if strings.Contains(err.Error(), "close sent") {
					fmt.Println("odje?")
					msg, err := TemplString(components.EndGameModal("1-0", "white"))
					if err != nil {
						respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
						return
					}
					err = onlineGame["white"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
					if err != nil {
						respondWithAnError(w, http.StatusInternalServerError, "writing online message error: ", err)
						return
					}
				} else {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
				}
			}
		}
	} else {
		_, err := fmt.Fprint(w, message)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}

	cfg.Matches[currentGame] = match

	if match.isWhiteTurn && (match.whiteTimer < 0 || match.whiteTimer == 0) {
		msg, err := TemplString(components.EndGameModal("0-1", "black"))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
			return
		}
		if found {
			err := onlineGame["white"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "writing online message error: ", err)
				return
			}
			err = onlineGame["black"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "writing online message error: ", err)
				return
			}
			return
		} else {
			_, err := fmt.Fprint(w, msg)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
	} else if !match.isWhiteTurn && (match.blackTimer < 0 || match.blackTimer == 0) {
		msg, err := TemplString(components.EndGameModal("1-0", "white"))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
			return
		}
		if found {
			err := onlineGame["white"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "writing online message error: ", err)
				return
			}
			err = onlineGame["black"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "writing online message error: ", err)
				return
			}
			return
		} else {
			_, err := fmt.Fprint(w, msg)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
	}
	log.Println(match.disconnected, "je li")
	select {
	case disconnected := <-match.disconnected:
		log.Println("jeste bgm", disconnected)
	default:
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

	message := fmt.Sprintf(`
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
						<img src="/assets/pieces/%v.svg" />
					</span>

					<div id="overlay" hx-swap-oob="true" class="hidden w-board w-board-md h-board h-board-md absolute z-20 hover:cursor-default"></div>

					<div id="promotion" hx-swap-oob="true" class="absolute"></div>
				`,
		pawnName,
		currentSquare.Coordinates[0],
		currentSquare.Coordinates[1],
		currentSquare.Piece.Image,
	)
	if found {
		for playerColor, onlinePlayer := range onlineGame {
			err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
				return
			}
		}
	} else {
		_, err := fmt.Fprint(w, message)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
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

	noCheck, err := handleIfCheck(w, cfg, newPiece, c.Value)
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
			cfg.endTurn(c.Value, r, w)
			return
		}

		currentGame.isWhiteUnderCheck = false
		currentGame.isBlackUnderCheck = false
		currentGame.tilesUnderAttack = []string{}
		getKing := currentGame.pieces[kingName]
		getKingSquare := currentGame.board[getKing.Tile]

		message := fmt.Sprintf(`
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			getKing.Name,
			getKingSquare.Coordinates[0],
			getKingSquare.Coordinates[1],
			getKing.Image,
		)
		if found {
			for playerColor, onlinePlayer := range onlineGame {
				err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
					return
				}
			}
		} else {
			_, err := fmt.Fprint(w, message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
	}

	currentGame.possibleEnPessant = ""
	currentGame.movesSinceLastCapture++
	cfg.Matches[c.Value] = currentGame
	cfg.endTurn(c.Value, r, w)
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
		err := match["white"].Conn.Close()
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "closing the connection error", err)
			return
		}
		err = match["black"].Conn.Close()
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "closing the connection error", err)
			return
		}
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
		if connection["white"].ID == userId {
			msg, err = TemplString(components.EndGameModal("0-1", "black"))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
				return
			}
		} else if connection["black"].ID == userId {
			msg, err = TemplString(components.EndGameModal("1-0", "white"))
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error converting component to string", err)
				return
			}
		}
		err = connection["white"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "writing online message error", err)
			return
		}
		err = connection["black"].Conn.WriteMessage(websocket.TextMessage, []byte(msg))
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
