package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/a-h/templ"
	"github.com/google/uuid"
)

var mockBoard = [][]string{
	{"8a", "8b", "8c", "8d", "8e", "8f", "8g", "8h"},
	{"7a", "7b", "7c", "7d", "7e", "7f", "7g", "7h"},
	{"6a", "6b", "6c", "6d", "6e", "6f", "6g", "6h"},
	{"5a", "5b", "5c", "5d", "5e", "5f", "5g", "5h"},
	{"4a", "4b", "4c", "4d", "4e", "4f", "4g", "4h"},
	{"3a", "3b", "3c", "3d", "3e", "3f", "3g", "3h"},
	{"2a", "2b", "2c", "2d", "2e", "2f", "2g", "2h"},
	{"1a", "1b", "1c", "1d", "1e", "1f", "1g", "1h"},
}

var rowIdxMap = map[string]int{
	"8": 0,
	"7": 1,
	"6": 2,
	"5": 3,
	"4": 4,
	"3": 5,
	"2": 6,
	"1": 7,
}

func canPlay(piece components.Piece, match Match, onlineGame map[string]components.OnlinePlayerStruct, userId uuid.UUID) bool {
	if onlineGame != nil {

		if userId == uuid.Nil {
			return false
		}

		if piece.IsWhite && match.isWhiteTurn && onlineGame["white"].ID == userId {
			return true
		} else if match.selectedPiece.IsWhite && piece.IsWhite && match.isWhiteTurn && onlineGame["white"].ID == userId {
			return true
		} else if !piece.IsWhite && !match.isWhiteTurn && onlineGame["black"].ID == userId {
			return true
		} else if !match.selectedPiece.IsWhite && !piece.IsWhite && !match.isWhiteTurn && onlineGame["black"].ID == userId {
			return true
		}

		return false
	}
	if match.isWhiteTurn {
		if piece.IsWhite {
			return true
		} else if match.selectedPiece.IsWhite && piece.IsWhite {
			return true
		}
	} else {
		if !piece.IsWhite {
			return true
		} else if !match.selectedPiece.IsWhite && !piece.IsWhite {
			return true
		}
	}

	return false
}

func canEat(selectedPiece, currentPiece components.Piece) bool {
	if (selectedPiece.IsWhite &&
		!currentPiece.IsWhite) ||
		(!selectedPiece.IsWhite &&
			currentPiece.IsWhite) {
		return true
	}

	return false
}

func fillBoard(match Match) Match {
	for _, v := range match.pieces {
		getTile := match.board[v.Tile]
		getTile.Piece = v
		match.board[v.Tile] = getTile
	}

	return match
}

func checkLegalMoves(match Match) []string {
	var startingPosition [2]int

	var possibleMoves []string

	if match.selectedPiece.Tile == "" {
		return possibleMoves
	}

	rowIdx := rowIdxMap[string(match.selectedPiece.Tile[0])]

	for i := 0; i < len(mockBoard[rowIdx]); i++ {
		if mockBoard[rowIdx][i] == match.selectedPiece.Tile {
			startingPosition = [2]int{rowIdx, i}
			break
		}
	}

	var pieceColor string

	if match.selectedPiece.IsWhite {
		pieceColor = "white"
	} else {
		pieceColor = "black"
	}

	if match.selectedPiece.IsPawn {
		getPawnMoves(&possibleMoves, startingPosition, match)
	} else {
		for _, move := range match.selectedPiece.LegalMoves {
			getMoves(&possibleMoves, startingPosition, move, match.selectedPiece.MovesOnce, pieceColor, match)
		}
	}

	return possibleMoves
}

func getPawnMoves(possible *[]string, startingPosition [2]int, match Match) {
	var moveIndex int
	piece := match.selectedPiece
	if piece.IsWhite {
		moveIndex = -1
	} else {
		moveIndex = 1
	}
	currentPosition := [2]int{startingPosition[0] + moveIndex, startingPosition[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return
	}

	if startingPosition[1]+1 < len(mockBoard[startingPosition[0]]) {
		currentTile := mockBoard[currentPosition[0]][startingPosition[1]+1]
		pieceOnCurrentTile := match.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		} else if strings.Contains(match.possibleEnPessant, currentTile) {
			*possible = append(*possible, fmt.Sprintf("enpessant_%v", currentTile))
		}
	}

	if startingPosition[1]-1 >= 0 {
		currentTile := mockBoard[currentPosition[0]][startingPosition[1]-1]
		pieceOnCurrentTile := match.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		} else if strings.Contains(match.possibleEnPessant, currentTile) {
			*possible = append(*possible, fmt.Sprintf("enpessant_%v", currentTile))
		}
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := match.board[currentTile].Piece

	if pieceOnCurrentTile.Name != "" {
		return
	}

	*possible = append(*possible, currentTile)

	if !piece.Moved {
		tile := mockBoard[currentPosition[0]+moveIndex][currentPosition[1]]
		pT := match.board[tile].Piece
		if pT.Name == "" {
			*possible = append(*possible, tile)
		}
	}
}

func getMoves(possible *[]string, startingPosition [2]int, move []int, checkOnce bool, pieceColor string, match Match) {
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := match.board[currentTile].Piece

	if pieceOnCurrentTile.Name != "" {
		if strings.Contains(pieceOnCurrentTile.Name, pieceColor) {
			return
		} else if !strings.Contains(pieceOnCurrentTile.Name, pieceColor) {
			*possible = append(*possible, currentTile)
			return
		}
	}

	*possible = append(*possible, currentTile)

	if checkOnce {
		return
	}

	getMoves(possible, currentPosition, move, checkOnce, pieceColor, match)
}

func samePiece(selectedPiece, currentPiece components.Piece) bool {
	if selectedPiece.IsWhite && currentPiece.IsWhite {
		return true
	} else if !selectedPiece.IsWhite && !currentPiece.IsWhite {
		return true
	}

	return false
}

func checkForCastle(match Match, currentPiece components.Piece) (bool, bool) {
	selectedPiece := match.selectedPiece
	b := match.board
	if (selectedPiece.IsKing &&
		strings.Contains(currentPiece.Name, "rook") ||
		(strings.Contains(selectedPiece.Name, "rook") &&
			currentPiece.IsKing)) &&
		!selectedPiece.Moved &&
		!currentPiece.Moved {

		var selectedStartingPosition [2]int
		var currentStartingPosition [2]int
		var tilesForCastle []string

		rowIdx := rowIdxMap[string(selectedPiece.Tile[0])]

		for i := 0; i < len(mockBoard[rowIdx]); i++ {
			if mockBoard[rowIdx][i] == selectedPiece.Tile {
				selectedStartingPosition = [2]int{rowIdx, i}
			}
			if mockBoard[rowIdx][i] == currentPiece.Tile {
				currentStartingPosition = [2]int{rowIdx, i}
			}
		}

		if selectedStartingPosition[1] > currentStartingPosition[1] {
			for i := range selectedStartingPosition[1] - currentStartingPosition[1] - 1 {
				getSquare := mockBoard[selectedStartingPosition[0]][selectedStartingPosition[1]-i-1]
				tilesForCastle = append(tilesForCastle, getSquare)
				pieceOnSquare := b[getSquare]
				if pieceOnSquare.Piece.Name != "" {
					return false, false
				}
			}
		}
		if selectedStartingPosition[1] < currentStartingPosition[1] {
			for i := range currentStartingPosition[1] - selectedStartingPosition[1] - 1 {
				getSquare := mockBoard[selectedStartingPosition[0]][currentStartingPosition[1]-i-1]
				tilesForCastle = append(tilesForCastle, getSquare)
				pieceOnSquare := b[getSquare]
				if pieceOnSquare.Piece.Name != "" {
					return false, false
				}
			}
		}

		var kingCheck bool
		if slices.ContainsFunc(tilesForCastle, func(tile string) bool {
			return handleChecksWhenKingMoves(tile, match)
		}) {
			kingCheck = true
		}

		if kingCheck {
			return true, true
		}

		return true, false
	}

	return false, false
}

func (cfg *appConfig) handleCheckForCheck(currentSquareName, currentGame string, selectedPiece components.Piece) (bool, components.Piece, []string) {
	match := cfg.Matches[currentGame]
	var startingPosition [2]int

	var king components.Piece
	var pieceColor string

	savedStartingTile := selectedPiece.Tile
	savedStartSqua := match.board[savedStartingTile]
	saved := match.board[currentSquareName]

	if currentSquareName != "" {
		startingSquare := match.board[selectedPiece.Tile]
		startingSquare.Piece = components.Piece{}
		match.board[selectedPiece.Tile] = startingSquare
		selectedPiece.Tile = currentSquareName
		curSq := match.board[currentSquareName]
		curSq.Piece = selectedPiece
		match.board[currentSquareName] = curSq

		var kingName string
		if selectedPiece.IsWhite {
			kingName = "white_king"
			pieceColor = "white"
		} else {
			kingName = "black_king"
			pieceColor = "black"
		}

		king = match.pieces[kingName]
	} else {
		var kingName string
		if selectedPiece.IsWhite {
			kingName = "black_king"
			pieceColor = "black"
		} else {
			kingName = "white_king"
			pieceColor = "white"
		}
		king = match.pieces[kingName]
	}

	rowIdx := rowIdxMap[string(king.Tile[0])]

	for i := 0; i < len(mockBoard[rowIdx]); i++ {
		if mockBoard[rowIdx][i] == king.Tile {
			startingPosition = [2]int{rowIdx, i}
			break
		}
	}

	kingLegalMoves := [][]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}, {2, 1}, {2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}, {-2, 1}, {-2, -1}}

	var tilesComb []string

	var check bool

	for _, move := range kingLegalMoves {
		var tilesUnderCheck []string
		checkInFor := checkCheck(&tilesUnderCheck, startingPosition, startingPosition, move, pieceColor, match)
		if checkInFor {
			check = true
			if len(tilesComb) == 0 {
				tilesComb = tilesUnderCheck
			} else {
				var tc []string
				for _, t := range tilesUnderCheck {
					if slices.Contains(tilesComb, t) {
						tc = append(tc, t)
					}
				}
				tilesComb = tc
			}
		}
	}

	if check {
		match.board[savedStartingTile] = savedStartSqua
		match.board[currentSquareName] = saved
		selectedPiece.Tile = savedStartingTile
		cfg.Matches[currentGame] = match

		return check, king, tilesComb
	}

	return false, king, []string{}
}

func checkCheck(tilesUnderCheck *[]string, startingPosition, startPosCompare [2]int, move []int, pieceColor string, match Match) bool {
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return false
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return false
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := match.board[currentTile].Piece

	if pieceOnCurrentTile.Name != "" {
		if strings.Contains(pieceOnCurrentTile.Name, pieceColor) {
			return false
		} else if !strings.Contains(pieceOnCurrentTile.Name, pieceColor) &&
			strings.Contains(pieceOnCurrentTile.Image, "knight") {
			for _, mv := range pieceOnCurrentTile.LegalMoves {
				if (mv[0] == move[0] && mv[1] == move[1]) && startPosCompare[0] == startingPosition[0] && startPosCompare[1] == startingPosition[1] {
					*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
					return true
				}
			}
			return false
		} else if !strings.Contains(pieceOnCurrentTile.Name, pieceColor) &&
			pieceOnCurrentTile.IsPawn {
			if pieceColor == "white" && ((move[0] == -1 && (move[1] == 1 || move[1] == -1)) && startPosCompare[0] == startingPosition[0] && startPosCompare[1] == startingPosition[1]) {
				*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
				return true
			} else if pieceColor == "black" && ((move[0] == 1 && (move[1] == 1 || move[1] == -1)) && startPosCompare[0] == startingPosition[0] && startPosCompare[1] == startingPosition[1]) {
				*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
				return true
			} else {
				return false
			}
		} else if !strings.Contains(pieceOnCurrentTile.Name, pieceColor) &&
			pieceOnCurrentTile.IsKing {
			for _, mv := range pieceOnCurrentTile.LegalMoves {
				if (mv[0] == move[0] && mv[1] == move[1]) && startPosCompare[0] == startingPosition[0] && startPosCompare[1] == startingPosition[1] {
					*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
					return true
				}
			}
			return false
		} else if !strings.Contains(pieceOnCurrentTile.Name, pieceColor) {
			for _, mv := range pieceOnCurrentTile.LegalMoves {
				if mv[0] == move[0] && mv[1] == move[1] {
					*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
					return true
				}
			}
			return false
		}
	}

	check := checkCheck(tilesUnderCheck, currentPosition, startPosCompare, move, pieceColor, match)
	if check {
		*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
	}

	return check
}

func handleChecksWhenKingMoves(currentSquareName string, match Match) bool {
	var kingPosition [2]int
	var king components.Piece
	var pieceColor string

	if match.isWhiteTurn {
		king = match.pieces["white_king"]
		pieceColor = "white"
	} else {
		king = match.pieces["black_king"]
		pieceColor = "black"
	}

	savedStartingTile := king.Tile
	savedStartSqua := match.board[savedStartingTile]
	saved := match.board[currentSquareName]

	startingSquare := match.board[king.Tile]
	startingSquare.Piece = components.Piece{}
	match.board[king.Tile] = startingSquare
	king.Tile = currentSquareName
	curSq := match.board[currentSquareName]
	curSq.Piece = king
	match.board[currentSquareName] = curSq

	rowIdx := rowIdxMap[string(king.Tile[0])]

	for i := 0; i < len(mockBoard[rowIdx]); i++ {
		if mockBoard[rowIdx][i] == king.Tile {
			kingPosition = [2]int{rowIdx, i}
			break
		}
	}

	kingLegalMoves := [][]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}, {2, 1}, {2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}, {-2, 1}, {-2, -1}}

	for _, move := range kingLegalMoves {
		var tilesUnderCheck []string
		if checkCheck(&tilesUnderCheck, kingPosition, kingPosition, move, pieceColor, match) {
			match.board[savedStartingTile] = savedStartSqua
			match.board[currentSquareName] = saved
			king.Tile = savedStartingTile
			return true
		}
	}

	match.board[savedStartingTile] = savedStartSqua
	match.board[currentSquareName] = saved
	king.Tile = savedStartingTile

	return false
}

func (cfg *appConfig) gameDone(match Match, currentGame string, w http.ResponseWriter) {
	var king components.Piece
	if match.isWhiteTurn {
		king = match.pieces["white_king"]
	} else {
		king = match.pieces["black_king"]
	}

	onlineGame, found := cfg.connections[currentGame]

	if match.movesSinceLastCapture == 50 {
		msg, err := TemplString(components.EndGameModal("1-1", ""))
		if err != nil {
			logError("couldn't render end game modal", err)
			return
		}
		err = sendMessage(onlineGame, found, w, msg, [2][]int{})
		if err != nil {
			logError("couldn't render end game modal", err)
			return
		}
		return
	}

	savePiece := match.selectedPiece
	match.selectedPiece = king
	legalMoves := checkLegalMoves(match)
	match.selectedPiece = savePiece
	var checkCount []bool
	for _, move := range legalMoves {
		if handleChecksWhenKingMoves(move, match) {
			checkCount = append(checkCount, true)
		}
	}
	if len(legalMoves) == len(checkCount) {
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			for _, piece := range match.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					legalMoves := checkLegalMoves(match)
					match.selectedPiece = savePiece

					for _, move := range legalMoves {
						if slices.Contains(match.tilesUnderAttack, move) {
							return
						}
					}
				}
			}
			msg, err := TemplString(components.EndGameModal("0-1", "black"))
			if err != nil {
				logError("couldn't convert component to string", err)
				return
			}
			err = sendMessage(onlineGame, found, w, msg, [2][]int{})
			if err != nil {
				logError("couldn't render end game modal", err)
				return
			}
		} else if !match.isWhiteTurn && match.isBlackUnderCheck {
			for _, piece := range match.pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					legalMoves := checkLegalMoves(match)
					match.selectedPiece = savePiece

					for _, move := range legalMoves {
						if slices.Contains(match.tilesUnderAttack, move) {
							return
						}
					}
				}
			}
			msg, err := TemplString(components.EndGameModal("1-0", "white"))
			if err != nil {
				logError("couldn't convert component to string", err)
				return
			}
			err = sendMessage(onlineGame, found, w, msg, [2][]int{})
			if err != nil {
				logError("couldn't render end game modal", err)
				return
			}
		} else if match.isWhiteTurn {
			for _, piece := range match.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					legalMoves := checkLegalMoves(match)
					match.selectedPiece = savePiece

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			msg, err := TemplString(components.EndGameModal("1-1", ""))
			if err != nil {
				logError("couldn't convert component to string", err)
				return
			}
			err = sendMessage(onlineGame, found, w, msg, [2][]int{})
			if err != nil {
				logError("couldn't render end game modal", err)
				return
			}
		} else if !match.isWhiteTurn {
			for _, piece := range match.pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					legalMoves := checkLegalMoves(match)
					match.selectedPiece = savePiece

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			msg, err := TemplString(components.EndGameModal("1-1", ""))
			if err != nil {
				logError("couldn't convert component to string", err)
				return
			}
			err = sendMessage(onlineGame, found, w, msg, [2][]int{})
			if err != nil {
				logError("couldn't render end game modal", err)
				return
			}
		}
	} else {
		return
	}
}

func setUserCheck(king components.Piece, currentMatch *Match) {
	if king.IsWhite {
		currentMatch.isWhiteUnderCheck = true
	} else {
		currentMatch.isBlackUnderCheck = true
	}
}

func handleIfCheck(w http.ResponseWriter, r *http.Request, cfg *appConfig, selected components.Piece, currentGame string) (bool, error) {
	match := cfg.Matches[currentGame]
	check, king, tilesUnderAttack := cfg.handleCheckForCheck("", currentGame, selected)
	kingSquare := match.board[king.Tile]
	if check {
		setUserCheck(king, &match)
		err := cfg.respondWithCheck(w, kingSquare, king, currentGame)
		if err != nil {
			return false, err
		}
		match.tilesUnderAttack = tilesUnderAttack
		cfg.Matches[currentGame] = match
		for _, tile := range tilesUnderAttack {
			t := match.board[tile]

			if t.Piece.Name != "" {
				err := respondWithNewPiece(w, r, t)

				if err != nil {
					return false, err
				}
			} else {
				err := cfg.respondWithCoverCheck(w, tile, t, currentGame)
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return true, nil
}

func bigCleanup(currentSquareName string, match *Match) {
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile
	selSeq := match.board[selectedSquare]
	currentSquare.Selected = false
	currentPiece := match.pieces[match.selectedPiece.Name]
	currentPiece.Tile = currentSquareName
	currentPiece.Moved = true
	match.pieces[match.selectedPiece.Name] = currentPiece
	currentSquare.Piece = currentPiece
	match.selectedPiece = components.Piece{}
	selSeq.Piece = match.selectedPiece
	match.board[selectedSquare] = selSeq
	match.board[currentSquareName] = currentSquare
}

func formatTime(seconds int) string {
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func (cfg *appConfig) endTurn(currentGame string, w http.ResponseWriter) {
	match := cfg.Matches[currentGame]
	if match.isWhiteTurn {
		match.whiteTimer += match.addition
	} else {
		match.blackTimer += match.addition
	}
	match.isWhiteTurn = !match.isWhiteTurn
	cfg.Matches[currentGame] = match
	cfg.gameDone(match, currentGame, w)
}

func (cfg *appConfig) refreshToken(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("refresh_token")

	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "refresh token not found", err)
		return
	}

	dbToken, err := cfg.database.SearchForToken(r.Context(), c.Value)

	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "refresh token not found", err)
		return
	}

	if dbToken.ExpiresAt.Before(time.Now()) {
		delete(cfg.users, dbToken.UserID)
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	user, err := cfg.database.GetUserById(r.Context(), dbToken.UserID)

	if err != nil {
		logError("no user with that id", err)
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	newToken, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't make jwt", err)
		return
	}

	newC := http.Cookie{
		Name:     "access_token",
		Value:    newToken,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &newC)

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) checkUser(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie("access_token")

	if err != nil {
		return err
	} else if c.Value != "" {
		userId, err := auth.ValidateJWT(c.Value, cfg.secret)

		if err != nil {
			if strings.Contains(err.Error(), "token is expired") {
				return err
			}
			return err
		} else if userId != uuid.Nil {
			_, err := cfg.database.GetUserById(r.Context(), userId)

			if err != nil {
				return err
			}

			_, ok := cfg.users[userId]

			if ok {
				http.Redirect(w, r, "/private", http.StatusSeeOther)
			}
		}
	}
	return nil
}

func (cfg *appConfig) checkUserPrivate(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie("access_token")
	if err != nil {
		return err
	} else if c.Value != "" {
		userId, err := auth.ValidateJWT(c.Value, cfg.secret)

		if err != nil {
			logError("user not found", err)
			http.Redirect(w, r, "/", http.StatusFound)
		} else if userId != uuid.Nil {
			_, err := cfg.database.GetUserById(r.Context(), userId)
			if err != nil {
				return err
			}
			_, ok := cfg.users[userId]
			if !ok {
				http.Redirect(w, r, "/", http.StatusFound)
			}
		}
	}
	return nil
}

func (cfg *appConfig) middleWareCheckForUser(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := cfg.checkUser(w, r)
		if err != nil {
			if !strings.Contains(err.Error(), "named cookie not present") {
				logError("error with check user", err)
			}
		}
		next(w, r)
	})
}

func (cfg *appConfig) middleWareCheckForUserPrivate(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := cfg.checkUserPrivate(w, r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		next(w, r)
	})
}

func (cfg *appConfig) isUserLoggedIn(r *http.Request) (uuid.UUID, error) {
	c, err := r.Cookie("access_token")

	if err != nil {
		return uuid.Nil, err
	}

	userId, err := auth.ValidateJWT(c.Value, cfg.secret)

	if err != nil {
		return uuid.Nil, err
	}

	_, err = cfg.database.GetUserById(r.Context(), userId)
	if err != nil {
		return uuid.Nil, err
	}
	_, ok := cfg.users[userId]
	if !ok {
		return uuid.Nil, err
	}

	return userId, nil
}

func (cfg *appConfig) showMoves(match Match, squareName, pieceName string, w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie("current_game")
	if err != nil {
		return err
	}
	if c.Value != "" {
		if strings.Split(c.Value, ":")[0] == "database" {
			return err
		}
	}
	onlineGame, found := cfg.connections[c.Value]
	boardState := make(map[string]string, 0)
	for k, v := range match.pieces {
		boardState[k] = v.Tile
	}

	jsonBoard, err := json.Marshal(boardState)

	if err != nil {
		return err
	}

	userId, err := cfg.isUserLoggedIn(r)
	if err != nil && !strings.Contains(err.Error(), "named cookie not present") {
		return err
	}

	if userId != uuid.Nil {
		err = cfg.database.CreateMove(r.Context(), database.CreateMoveParams{
			Board:     jsonBoard,
			Move:      fmt.Sprintf("%v:%v", pieceName, squareName),
			WhiteTime: int32(match.whiteTimer),
			BlackTime: int32(match.blackTimer),
			MatchID:   match.matchId,
		})

		if err != nil {
			return err
		}
	}

	var message string
	if len(match.allMoves)%2 == 0 {
		message = fmt.Sprintf(
			getMovesUpdateMessage(),
			squareName,
		)
	} else {
		message = fmt.Sprintf(
			getMovesNumberUpdateMessage(),
			len(match.allMoves)/2+1,
			squareName,
		)
	}

	cfg.Matches[c.Value] = match

	err = sendMessage(onlineGame, found, w, message, [2][]int{})

	return err
}

func cleanFillBoard(match Match, pieces map[string]components.Piece) Match {
	match.pieces = pieces
	for i, tile := range match.board {
		tile.Piece = components.Piece{}
		match.board[i] = tile
	}
	for _, v := range pieces {
		getTile := match.board[v.Tile]
		getTile.Piece = v
		match.board[v.Tile] = getTile
	}
	return match
}

func (cfg *appConfig) checkForPawnPromotion(pawnName, currentGame string, w http.ResponseWriter, r *http.Request) (bool, error) {
	var isOnLastTile bool
	match := cfg.Matches[currentGame]
	onlineGame, found := cfg.connections[currentGame]
	pawn := match.pieces[pawnName]
	if !pawn.IsPawn {
		return false, nil
	}
	square := match.board[pawn.Tile]
	var pieceColor string
	var firstPosition string
	if pawn.IsWhite {
		pieceColor = "white"
		firstPosition = "top: 0px"
	} else {
		pieceColor = "black"
		firstPosition = "bottom: 0px"
	}

	var multiplier int

	if found {
		userC, err := r.Cookie("access_token")

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
			return false, err
		}

		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusUnauthorized, "user not found")
			return false, err
		}

		for _, player := range onlineGame.players {
			if player.ID == userId {
				multiplier = player.Multiplier
			}
		}
	} else {
		multiplier = match.coordinateMultiplier
	}
	endBoardCoordinates := 7 * multiplier
	dropdownPosition := square.Coordinates[1] + multiplier
	if square.Coordinates[1] == endBoardCoordinates {
		dropdownPosition = square.Coordinates[1] - multiplier
	}

	rowIdx := rowIdxMap[string(pawn.Tile[0])]
	if pawn.IsWhite && rowIdx == 0 || !pawn.IsWhite && rowIdx == 7 {
		isOnLastTile = true
		message := fmt.Sprintf(
			getPromotionInitMessage(),
			firstPosition,
			dropdownPosition,
			pieceColor,
			pawnName,
			pieceColor,
			pieceColor,
			pawnName,
			pieceColor,
			pieceColor,
			pawnName,
			pieceColor,
			pieceColor,
			pawnName,
			pieceColor,
		)

		_, err := fmt.Fprint(w, message)

		if err != nil {
			return false, err
		}
	}
	return isOnLastTile, nil
}

func TemplString(t templ.Component) (string, error) {
	var b bytes.Buffer
	if err := t.Render(context.Background(), &b); err != nil {
		return "", err
	}
	return b.String(), nil
}

func checkForEnPessant(selectedSquare string, currentSquare components.Square, match Match) Match {
	if match.selectedPiece.IsPawn && !match.selectedPiece.Moved {
		if match.board[selectedSquare].CoordinatePosition[0]-currentSquare.CoordinatePosition[0] == 2 {
			freeTile := mockBoard[currentSquare.CoordinatePosition[0]-2][currentSquare.CoordinatePosition[1]]
			match.possibleEnPessant = fmt.Sprintf("white_%v", freeTile)
		} else if currentSquare.CoordinatePosition[0]-match.board[selectedSquare].CoordinatePosition[0] == 2 {
			freeTile := mockBoard[currentSquare.CoordinatePosition[0]+2][currentSquare.CoordinatePosition[1]]
			match.possibleEnPessant = fmt.Sprintf("black_%v", freeTile)
		}
	}
	return match
}

func replaceStyles(text string, bottom, left []int) string {
	re := regexp.MustCompile(`style="bottom:\s*[\d.]+px;\s*left:\s*[\d.]+px"`)

	replacements := []string{}

	for i := range bottom {
		replacements = append(replacements, fmt.Sprintf(`style="bottom: %vpx; left: %vpx"`, bottom[i], left[i]))
	}

	matches := re.FindAllStringIndex(text, -1)

	var builder strings.Builder
	lastIndex := 0

	for i, match := range matches {
		start, end := match[0], match[1]

		builder.WriteString(text[lastIndex:start])

		if i < len(replacements) {
			builder.WriteString(replacements[i])
		} else {
			builder.WriteString(text[start:end])
		}

		lastIndex = end
	}

	builder.WriteString(text[lastIndex:])

	output := builder.String()

	return output
}

func eatCleanup(match Match, pieceToDelete components.Piece, squareToDeleteName, currentSquareName string) (Match, components.Square, components.Piece) {
	squareToDelete := match.board[squareToDeleteName]
	currentSquare := match.board[currentSquareName]

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

	return match, squareToDelete, saveSelected
}

func sendMessage(onlineGame OnlineGame, found bool, w http.ResponseWriter, msg string, args [2][]int) error {
	if found && len(args) > 0 {
		for _, onlinePlayer := range onlineGame.players {
			var bottomCoordinates []int
			for _, coordinate := range args[0] {
				bottomCoordinates = append(bottomCoordinates, coordinate*onlinePlayer.Multiplier)
			}
			var leftCoordinates []int
			for _, coordinate := range args[1] {
				leftCoordinates = append(leftCoordinates, coordinate*onlinePlayer.Multiplier)
			}

			newMessage := replaceStyles(msg, bottomCoordinates, leftCoordinates)
			onlineGame.playerMsg <- newMessage
			onlineGame.player <- onlinePlayer
		}

	} else if found {
		onlineGame.message <- msg
	} else {
		_, err := fmt.Fprint(w, msg)
		if err != nil {
			return err
		}
	}
	return nil
}
