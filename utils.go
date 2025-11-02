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

func (m *Match) canPlay(piece components.Piece, onlineGame map[string]components.OnlinePlayerStruct, userId uuid.UUID) bool {
	if onlineGame != nil {

		if userId == uuid.Nil {
			return false
		}

		if piece.IsWhite && m.isWhiteTurn && onlineGame["white"].ID == userId {
			return true
		} else if m.selectedPiece.IsWhite && piece.IsWhite && m.isWhiteTurn && onlineGame["white"].ID == userId {
			return true
		} else if !piece.IsWhite && !m.isWhiteTurn && onlineGame["black"].ID == userId {
			return true
		} else if !m.selectedPiece.IsWhite && !piece.IsWhite && !m.isWhiteTurn && onlineGame["black"].ID == userId {
			return true
		}

		return false
	}
	if m.isWhiteTurn {
		if piece.IsWhite {
			return true
		} else if m.selectedPiece.IsWhite && piece.IsWhite {
			return true
		}
	} else {
		if !piece.IsWhite {
			return true
		} else if !m.selectedPiece.IsWhite && !piece.IsWhite {
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

func (m *Match) fillBoard() {
	for _, v := range m.pieces {
		getTile := m.board[v.Tile]
		getTile.Piece = v
		m.board[v.Tile] = getTile
	}
}

func (m *Match) checkLegalMoves() []string {
	var startingPosition [2]int

	var possibleMoves []string

	if m.selectedPiece.Tile == "" {
		return possibleMoves
	}

	rowIdx := rowIdxMap[string(m.selectedPiece.Tile[0])]

	for i := 0; i < len(mockBoard[rowIdx]); i++ {
		if mockBoard[rowIdx][i] == m.selectedPiece.Tile {
			startingPosition = [2]int{rowIdx, i}
			break
		}
	}

	var pieceColor string

	if m.selectedPiece.IsWhite {
		pieceColor = "white"
	} else {
		pieceColor = "black"
	}

	if m.selectedPiece.IsPawn {
		m.getPawnMoves(&possibleMoves, startingPosition)
	} else {
		for _, move := range m.selectedPiece.LegalMoves {
			m.getMoves(&possibleMoves, startingPosition, move, m.selectedPiece.MovesOnce, pieceColor)
		}
	}

	return possibleMoves
}

func (m *Match) getPawnMoves(possible *[]string, startingPosition [2]int) {
	var moveIndex int
	piece := m.selectedPiece
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
		pieceOnCurrentTile := m.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		} else if strings.Contains(m.possibleEnPessant, currentTile) {
			*possible = append(*possible, fmt.Sprintf("enpessant_%v", currentTile))
		}
	}

	if startingPosition[1]-1 >= 0 {
		currentTile := mockBoard[currentPosition[0]][startingPosition[1]-1]
		pieceOnCurrentTile := m.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		} else if strings.Contains(m.possibleEnPessant, currentTile) {
			*possible = append(*possible, fmt.Sprintf("enpessant_%v", currentTile))
		}
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := m.board[currentTile].Piece

	if pieceOnCurrentTile.Name != "" {
		return
	}

	*possible = append(*possible, currentTile)

	if !piece.Moved {
		tile := mockBoard[currentPosition[0]+moveIndex][currentPosition[1]]
		pT := m.board[tile].Piece
		if pT.Name == "" {
			*possible = append(*possible, tile)
		}
	}
}

func (m *Match) getMoves(possible *[]string, startingPosition [2]int, move []int, checkOnce bool, pieceColor string) {
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := m.board[currentTile].Piece

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

	m.getMoves(possible, currentPosition, move, checkOnce, pieceColor)
}

func samePiece(selectedPiece, currentPiece components.Piece) bool {
	if selectedPiece.IsWhite && currentPiece.IsWhite {
		return true
	} else if !selectedPiece.IsWhite && !currentPiece.IsWhite {
		return true
	}

	return false
}

func (m *Match) checkForCastle(currentPiece components.Piece) (bool, bool) {
	selectedPiece := m.selectedPiece
	b := m.board
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
			return m.handleChecksWhenKingMoves(tile)
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

func (m *Match) handleCheckForCheck(currentSquareName string, selectedPiece components.Piece) (bool, components.Piece, []string) {
	var startingPosition [2]int

	var king components.Piece
	var pieceColor string

	savedStartingTile := selectedPiece.Tile
	savedStartSqua := m.board[savedStartingTile]
	saved := m.board[currentSquareName]

	if currentSquareName != "" {
		startingSquare := m.board[selectedPiece.Tile]
		startingSquare.Piece = components.Piece{}
		m.board[selectedPiece.Tile] = startingSquare
		selectedPiece.Tile = currentSquareName
		curSq := m.board[currentSquareName]
		curSq.Piece = selectedPiece
		m.board[currentSquareName] = curSq

		var kingName string
		if selectedPiece.IsWhite {
			kingName = "white_king"
			pieceColor = "white"
		} else {
			kingName = "black_king"
			pieceColor = "black"
		}

		king = m.pieces[kingName]
	} else {
		var kingName string
		if selectedPiece.IsWhite {
			kingName = "black_king"
			pieceColor = "black"
		} else {
			kingName = "white_king"
			pieceColor = "white"
		}
		king = m.pieces[kingName]
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
		checkInFor := m.checkCheck(&tilesUnderCheck, startingPosition, startingPosition, move, pieceColor)
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
		m.board[savedStartingTile] = savedStartSqua
		m.board[currentSquareName] = saved
		selectedPiece.Tile = savedStartingTile

		return check, king, tilesComb
	}

	return false, king, []string{}
}

func (m *Match) checkCheck(tilesUnderCheck *[]string, startingPosition, startPosCompare [2]int, move []int, pieceColor string) bool {
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return false
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return false
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := m.board[currentTile].Piece

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

	check := m.checkCheck(tilesUnderCheck, currentPosition, startPosCompare, move, pieceColor)
	if check {
		*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
	}

	return check
}

func (m *Match) handleChecksWhenKingMoves(currentSquareName string) bool {
	var kingPosition [2]int
	var king components.Piece
	var pieceColor string

	if m.isWhiteTurn {
		king = m.pieces["white_king"]
		pieceColor = "white"
	} else {
		king = m.pieces["black_king"]
		pieceColor = "black"
	}

	savedStartingTile := king.Tile
	savedStartSqua := m.board[savedStartingTile]
	saved := m.board[currentSquareName]

	startingSquare := m.board[king.Tile]
	startingSquare.Piece = components.Piece{}
	m.board[king.Tile] = startingSquare
	king.Tile = currentSquareName
	curSq := m.board[currentSquareName]
	curSq.Piece = king
	m.board[currentSquareName] = curSq

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
		if m.checkCheck(&tilesUnderCheck, kingPosition, kingPosition, move, pieceColor) {
			m.board[savedStartingTile] = savedStartSqua
			m.board[currentSquareName] = saved
			king.Tile = savedStartingTile
			return true
		}
	}

	m.board[savedStartingTile] = savedStartSqua
	m.board[currentSquareName] = saved
	king.Tile = savedStartingTile

	return false
}

func (cfg *appConfig) gameDone(match Match, w http.ResponseWriter) {
	var king components.Piece
	if match.isWhiteTurn {
		king = match.pieces["white_king"]
	} else {
		king = match.pieces["black_king"]
	}

	onlineGame, found := match.isOnlineMatch()

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
	legalMoves := match.checkLegalMoves()
	match.selectedPiece = savePiece
	var checkCount []bool
	for _, move := range legalMoves {
		if match.handleChecksWhenKingMoves(move) {
			checkCount = append(checkCount, true)
		}
	}
	if len(legalMoves) == len(checkCount) {
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			for _, piece := range match.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					legalMoves := match.checkLegalMoves()
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
					legalMoves := match.checkLegalMoves()
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
					legalMoves := match.checkLegalMoves()
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
					legalMoves := match.checkLegalMoves()
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

func (m *Match) setUserCheck(king components.Piece) {
	if king.IsWhite {
		m.isWhiteUnderCheck = true
	} else {
		m.isBlackUnderCheck = true
	}
}

func handleIfCheck(w http.ResponseWriter, r *http.Request, cfg *appConfig, selected components.Piece, currentGame string) (bool, error) {
	match, _ := cfg.Matches.getMatch(currentGame)
	check, king, tilesUnderAttack := match.handleCheckForCheck("", selected)
	kingSquare := match.board[king.Tile]
	if check {
		match.setUserCheck(king)
		err := match.respondWithCheck(w, kingSquare, king)
		if err != nil {
			return false, err
		}
		match.tilesUnderAttack = tilesUnderAttack
		cfg.Matches.setMatch(currentGame, match)
		for _, tile := range tilesUnderAttack {
			t := match.board[tile]

			if t.Piece.Name != "" {
				err := respondWithNewPiece(w, r, t)

				if err != nil {
					return false, err
				}
			} else {
				err := match.respondWithCoverCheck(w, tile, t)
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return true, nil
}

func (m *Match) bigCleanup(currentSquareName string) {
	currentSquare := m.board[currentSquareName]
	selectedSquare := m.selectedPiece.Tile
	selSeq := m.board[selectedSquare]
	currentSquare.Selected = false
	currentPiece := m.pieces[m.selectedPiece.Name]
	currentPiece.Tile = currentSquareName
	currentPiece.Moved = true
	m.pieces[m.selectedPiece.Name] = currentPiece
	currentSquare.Piece = currentPiece
	m.selectedPiece = components.Piece{}
	selSeq.Piece = m.selectedPiece
	m.board[selectedSquare] = selSeq
	m.board[currentSquareName] = currentSquare
}

func formatTime(seconds int) string {
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func (cfg *appConfig) endTurn(currentGame string, w http.ResponseWriter) {
	match, _ := cfg.Matches.getMatch(currentGame)
	if match.isWhiteTurn {
		match.whiteTimer += match.addition
	} else {
		match.blackTimer += match.addition
	}
	match.isWhiteTurn = !match.isWhiteTurn
	cfg.Matches.setMatch(currentGame, match)
	cfg.gameDone(match, w)
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
	userId, err := cfg.getUserId(r)

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
	onlineGame, found := match.isOnlineMatch()
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

	cfg.Matches.setMatch(c.Value, match)

	err = sendMessage(onlineGame, found, w, message, [2][]int{})

	return err
}

func (m *Match) cleanFillBoard(pieces map[string]components.Piece) {
	m.pieces = pieces
	for i, tile := range m.board {
		tile.Piece = components.Piece{}
		m.board[i] = tile
	}
	for _, v := range pieces {
		getTile := m.board[v.Tile]
		getTile.Piece = v
		m.board[v.Tile] = getTile
	}
}

func (cfg *appConfig) checkForPawnPromotion(pawnName, currentGame string, w http.ResponseWriter, r *http.Request) (bool, error) {
	var isOnLastTile bool
	match, _ := cfg.Matches.getMatch(currentGame)
	onlineGame, found := match.isOnlineMatch()
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

func (m *Match) checkForEnPessant(selectedSquare string, currentSquare components.Square) {
	if m.selectedPiece.IsPawn && !m.selectedPiece.Moved {
		if m.board[selectedSquare].CoordinatePosition[0]-currentSquare.CoordinatePosition[0] == 2 {
			freeTile := mockBoard[currentSquare.CoordinatePosition[0]-2][currentSquare.CoordinatePosition[1]]
			m.possibleEnPessant = fmt.Sprintf("white_%v", freeTile)
		} else if currentSquare.CoordinatePosition[0]-m.board[selectedSquare].CoordinatePosition[0] == 2 {
			freeTile := mockBoard[currentSquare.CoordinatePosition[0]+2][currentSquare.CoordinatePosition[1]]
			m.possibleEnPessant = fmt.Sprintf("black_%v", freeTile)
		}
	}
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

func (m *Match) eatCleanup(pieceToDelete components.Piece, squareToDeleteName, currentSquareName string) (components.Square, components.Piece) {
	squareToDelete := m.board[squareToDeleteName]
	currentSquare := m.board[currentSquareName]

	m.allMoves = append(m.allMoves, currentSquareName)
	delete(m.pieces, pieceToDelete.Name)
	m.selectedPiece.Tile = currentSquareName
	m.pieces[m.selectedPiece.Name] = m.selectedPiece
	currentSquare.Piece = m.selectedPiece
	squareToDelete.Piece = components.Piece{}
	m.board[currentSquareName] = currentSquare
	m.board[squareToDeleteName] = squareToDelete
	saveSelected := m.selectedPiece
	m.selectedPiece = components.Piece{}
	m.possibleEnPessant = ""
	m.movesSinceLastCapture = 0

	return squareToDelete, saveSelected
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
