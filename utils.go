package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
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

func (cfg *appConfig) canPlay(piece components.Piece, currentGame string) bool {
	match := cfg.Matches[currentGame]
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

func (cfg *appConfig) fillBoard(currentGame string) {
	match := cfg.Matches[currentGame]
	for _, v := range match.pieces {
		getTile := match.board[v.Tile]
		getTile.Piece = v
		match.board[v.Tile] = getTile
	}
	cfg.Matches[currentGame] = match
}

func (cfg *appConfig) checkLegalMoves(currentGame string) []string {
	match := cfg.Matches[currentGame]
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
		cfg.getPawnMoves(&possibleMoves, startingPosition, match.selectedPiece, currentGame)
	} else {
		for _, move := range match.selectedPiece.LegalMoves {
			cfg.getMoves(&possibleMoves, startingPosition, move, match.selectedPiece.MovesOnce, pieceColor, currentGame)
		}
	}

	return possibleMoves
}

// TODO: IMPLEMENT EN PESSANT
func (cfg *appConfig) getPawnMoves(possible *[]string, startingPosition [2]int, piece components.Piece, currentGame string) {
	match := cfg.Matches[currentGame]
	var moveIndex int
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
		}
	}

	if startingPosition[1]-1 >= 0 {
		currentTile := mockBoard[currentPosition[0]][startingPosition[1]-1]
		pieceOnCurrentTile := match.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
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

func (cfg *appConfig) getMoves(possible *[]string, startingPosition [2]int, move []int, checkOnce bool, pieceColor, currentGame string) {
	match := cfg.Matches[currentGame]
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

	cfg.getMoves(possible, currentPosition, move, checkOnce, pieceColor, currentGame)
}

func samePiece(selectedPiece, currentPiece components.Piece) bool {
	if selectedPiece.IsWhite && currentPiece.IsWhite {
		return true
	} else if !selectedPiece.IsWhite && !currentPiece.IsWhite {
		return true
	}

	return false
}

func (cfg *appConfig) checkForCastle(b map[string]components.Square, selectedPiece, currentPiece components.Piece, currentGame string) (bool, bool) {
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
			return cfg.handleChecksWhenKingMoves(tile, currentGame)
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

func (cfg *appConfig) handleCastle(w http.ResponseWriter, currentPiece components.Piece, currentGame string, r *http.Request) error {
	match := cfg.Matches[currentGame]
	var king components.Piece
	var rook components.Piece

	if match.selectedPiece.IsKing {
		king = match.selectedPiece
		rook = currentPiece
	} else {
		king = currentPiece
		rook = match.selectedPiece
	}

	kTile := king.Tile
	rTile := rook.Tile
	savedKingTile := match.board[king.Tile]
	savedRookTile := match.board[rook.Tile]
	kingSquare := match.board[king.Tile]
	rookSquare := match.board[rook.Tile]

	if kingSquare.Coordinates[1] < rookSquare.Coordinates[1] {
		kC := kingSquare.Coordinates[1]
		rookSquare.Coordinates[1] = kC + match.coordinateMultiplier
		kingSquare.Coordinates[1] = kC + match.coordinateMultiplier*2
	} else {
		kC := kingSquare.Coordinates[1]
		rookSquare.Coordinates[1] = kC - match.coordinateMultiplier
		kingSquare.Coordinates[1] = kC - match.coordinateMultiplier*2
	}

	_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
		king.Name,
		kingSquare.Coordinates[0],
		kingSquare.Coordinates[1],
		king.Image,
		rook.Name,
		rookSquare.Coordinates[0],
		rookSquare.Coordinates[1],
		rook.Image,
	)
	if kingSquare.CoordinatePosition[1]-rookSquare.CoordinatePosition[1] == -3 {
		match.allMoves = append(match.allMoves, "O-O")
		cfg.showMoves(match, "O-O", "king", w, r)
	} else {
		match.allMoves = append(match.allMoves, "O-O-O")
		cfg.showMoves(match, "O-O-O", "king", w, r)
	}

	if err != nil {
		return err
	}

	rowIdx := rowIdxMap[string(king.Tile[0])]
	king.Tile = mockBoard[rowIdx][kingSquare.Coordinates[1]/match.coordinateMultiplier]
	rook.Tile = mockBoard[rowIdx][rookSquare.Coordinates[1]/match.coordinateMultiplier]
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
	cfg.Matches[currentGame] = match
	go cfg.gameDone(currentGame)

	return nil
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
		checkInFor := cfg.checkCheck(&tilesUnderCheck, startingPosition, startingPosition, move, pieceColor, currentGame)
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

func (cfg *appConfig) checkCheck(tilesUnderCheck *[]string, startingPosition, startPosCompare [2]int, move []int, pieceColor, currentGame string) bool {
	match := cfg.Matches[currentGame]
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
			strings.Contains(pieceOnCurrentTile.Name, "knight") {
			for _, mv := range pieceOnCurrentTile.LegalMoves {
				if (mv[0] == move[0] && mv[1] == move[1]) && startPosCompare[0] == startingPosition[0] && startPosCompare[1] == startingPosition[1] {
					*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
					return true
				}
			}
		} else if !strings.Contains(pieceOnCurrentTile.Name, pieceColor) &&
			pieceOnCurrentTile.IsPawn {
			if ((move[0] == 1 && (move[1] == 1 || move[1] == -1)) || (move[0] == -1 && (move[1] == 1 || move[1] == -1))) && startPosCompare[0] == startingPosition[0] && startPosCompare[1] == startingPosition[1] {
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

	check := cfg.checkCheck(tilesUnderCheck, currentPosition, startPosCompare, move, pieceColor, currentGame)
	if check {
		*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
	}

	return check
}

func (cfg *appConfig) handleChecksWhenKingMoves(currentSquareName, currentGame string) bool {
	match := cfg.Matches[currentGame]
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
		if cfg.checkCheck(&tilesUnderCheck, kingPosition, kingPosition, move, pieceColor, currentGame) {
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

func (cfg *appConfig) gameDone(currentGame string) {
	match := cfg.Matches[currentGame]
	var king components.Piece
	if match.isWhiteTurn {
		king = match.pieces["white_king"]
	} else {
		king = match.pieces["black_king"]
	}

	savePiece := match.selectedPiece
	match.selectedPiece = king

	cfg.Matches[currentGame] = match

	legalMoves := cfg.checkLegalMoves(currentGame)

	match.selectedPiece = savePiece
	cfg.Matches[currentGame] = match
	var checkCount []bool

	for _, move := range legalMoves {
		if cfg.handleChecksWhenKingMoves(move, currentGame) {
			checkCount = append(checkCount, true)
		}
	}

	if len(legalMoves) == len(checkCount) {
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			for _, piece := range match.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					cfg.Matches[currentGame] = match
					legalMoves := cfg.checkLegalMoves(currentGame)
					match.selectedPiece = savePiece
					cfg.Matches[currentGame] = match

					for _, move := range legalMoves {
						if slices.Contains(match.tilesUnderAttack, move) {
							return
						}
					}
				}
			}
			fmt.Println("checkmate")
		} else if !match.isWhiteTurn && match.isBlackUnderCheck {
			for _, piece := range match.pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					cfg.Matches[currentGame] = match
					legalMoves := cfg.checkLegalMoves(currentGame)
					match.selectedPiece = savePiece
					cfg.Matches[currentGame] = match

					for _, move := range legalMoves {
						if slices.Contains(match.tilesUnderAttack, move) {
							return
						}
					}
				}
			}
			fmt.Println("checkmate")
		} else if match.isWhiteTurn {
			for _, piece := range match.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					cfg.Matches[currentGame] = match
					legalMoves := cfg.checkLegalMoves(currentGame)
					match.selectedPiece = savePiece
					cfg.Matches[currentGame] = match

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			fmt.Println("stalemate")
		} else if !match.isWhiteTurn {
			for _, piece := range match.pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := match.selectedPiece
					match.selectedPiece = piece
					cfg.Matches[currentGame] = match
					legalMoves := cfg.checkLegalMoves(currentGame)
					match.selectedPiece = savePiece
					cfg.Matches[currentGame] = match

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			fmt.Println("stalemate")
		}
	} else {
		fmt.Println("you are good")
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

func handleIfCheck(w http.ResponseWriter, cfg *appConfig, selected components.Piece, currentGame string) bool {
	match := cfg.Matches[currentGame]
	check, king, tilesUnderAttack := cfg.handleCheckForCheck("", currentGame, selected)
	kingSquare := match.board[king.Tile]

	if check {
		setUserCheck(king, &match)
		err := respondWithCheck(w, kingSquare, king)

		if err != nil {
			fmt.Println(err)
		}

		match.tilesUnderAttack = tilesUnderAttack
		cfg.Matches[currentGame] = match

		for _, tile := range tilesUnderAttack {
			t := match.board[tile]

			if t.Piece.Name != "" {
				err := respondWithNewPiece(w, t)

				if err != nil {
					fmt.Println(err)
				}
			} else {
				err := respondWithCoverCheck(w, tile, t)

				if err != nil {
					fmt.Println(err)
				}
			}
		}
		return false
	}

	return true
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

func (cfg *appConfig) endTurn(w http.ResponseWriter, r *http.Request, currentGame string) {
	match := cfg.Matches[currentGame]
	if match.isWhiteTurn {
		match.whiteTimer += match.addition
	} else {
		match.blackTimer += match.addition
	}
	match.isWhiteTurn = !match.isWhiteTurn
	cfg.Matches[currentGame] = match
	cfg.gameDone(currentGame)
	cfg.timerHandler(w, r)
}

func (cfg *appConfig) refreshToken(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("refresh_token")

	if err != nil {
		fmt.Println(err)
		return
	}

	dbToken, err := cfg.database.SearchForToken(r.Context(), c.Value)

	if err != nil {
		fmt.Println(err)
		return
	}

	if dbToken.ExpiresAt.Before(time.Now()) {
		fmt.Println("token expired")
		delete(cfg.users, dbToken.UserID)
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	user, err := cfg.database.GetUserById(r.Context(), dbToken.UserID)

	if err != nil {
		fmt.Println("no user with that id")
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	newToken, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		fmt.Println("token err", err)
		w.WriteHeader(http.StatusInternalServerError)
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

func (cfg *appConfig) checkUser(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("access_token")

	if err != nil {
		fmt.Println(err)
	} else if c.Value != "" {
		userId, err := auth.ValidateJWT(c.Value, cfg.secret)

		if err != nil {
			if strings.Contains(err.Error(), "token is expired") {
				return
			}
			fmt.Println(err)
		} else if userId != uuid.Nil {
			_, err := cfg.database.GetUserById(r.Context(), userId)

			if err != nil {
				fmt.Println(err)
				return
			}

			_, ok := cfg.users[userId]

			if ok {
				http.Redirect(w, r, "/private", http.StatusSeeOther)
			}
		}
	}
}

func (cfg *appConfig) checkUserPrivate(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("access_token")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusFound)
	} else if c.Value != "" {
		userId, err := auth.ValidateJWT(c.Value, cfg.secret)

		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/", http.StatusFound)
		} else if userId != uuid.Nil {
			_, err := cfg.database.GetUserById(r.Context(), userId)
			if err != nil {
				fmt.Println(err)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			_, ok := cfg.users[userId]
			if !ok {
				fmt.Println("no user found")
				http.Redirect(w, r, "/", http.StatusFound)
			}
		}
	}
}

func (cfg *appConfig) middleWareCheckForUser(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.checkUser(w, r)
		next(w, r)
	})
}

func (cfg *appConfig) middleWareCheckForUserPrivate(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.checkUserPrivate(w, r)
		next(w, r)
	})
}

func (cfg *appConfig) isUserLoggedIn(r *http.Request) uuid.UUID {
	c, err := r.Cookie("access_token")

	if err != nil {
		fmt.Println("couldn't find the token")
		return uuid.Nil
	}

	userId, err := auth.ValidateJWT(c.Value, cfg.secret)

	if err != nil {
		fmt.Println("invalid token")
		return uuid.Nil
	}

	_, err = cfg.database.GetUserById(r.Context(), userId)
	if err != nil {
		fmt.Println(err)
		return uuid.Nil
	}
	_, ok := cfg.users[userId]
	if !ok {
		fmt.Println("no user found")
		return uuid.Nil
	}

	return userId
}

func (cfg *appConfig) showMoves(match Match, squareName, pieceName string, w http.ResponseWriter, r *http.Request) {

	boardState := make(map[string]string, 0)
	for k, v := range match.pieces {
		boardState[k] = v.Tile
	}

	jsonBoard, err := json.Marshal(boardState)

	if err != nil {
		fmt.Println(err)
		return
	}

	userId := cfg.isUserLoggedIn(r)

	if userId != uuid.Nil {
		err = cfg.database.CreateMove(r.Context(), database.CreateMoveParams{
			Board:     jsonBoard,
			Move:      fmt.Sprintf("%v:%v", pieceName, squareName),
			WhiteTime: int32(match.whiteTimer),
			BlackTime: int32(match.blackTimer),
			MatchID:   match.matchId,
		})

		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if len(match.allMoves)%2 == 0 {
		fmt.Fprintf(w, `
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v</span>
				</div>
			`,
			squareName,
		)
	} else {
		fmt.Fprintf(w, `
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v.</span>
					<span>%v</span>
				</div>
		`,
			len(match.allMoves)/2+1,
			squareName,
		)
	}
}

func (cfg *appConfig) cleanFillBoard(gameName string, pieces map[string]components.Piece) {
	match := cfg.Matches[gameName]
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
	cfg.Matches[gameName] = match
}
