package main

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
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

func (gcfg *gameConfig) canPlay(piece components.Piece, currentGame string) bool {
	cfg := gcfg.Matches[currentGame]
	if cfg.isWhiteTurn {
		if piece.IsWhite {
			return true
		} else if cfg.selectedPiece.IsWhite && piece.IsWhite {
			return true
		}
	} else {
		if !piece.IsWhite {
			return true
		} else if !cfg.selectedPiece.IsWhite && !piece.IsWhite {
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

func (gcfg *gameConfig) fillBoard(currentGame string) {
	cfg := gcfg.Matches[currentGame]
	for _, v := range cfg.pieces {
		getTile := cfg.board[v.Tile]
		getTile.Piece = v
		cfg.board[v.Tile] = getTile
	}
}

func (gcfg *gameConfig) checkLegalMoves(currentGame string) []string {
	cfg := gcfg.Matches[currentGame]
	var startingPosition [2]int

	var possibleMoves []string

	if cfg.selectedPiece.Tile == "" {
		return possibleMoves
	}

	rowIdx := rowIdxMap[string(cfg.selectedPiece.Tile[0])]

	for i := 0; i < len(mockBoard[rowIdx]); i++ {
		if mockBoard[rowIdx][i] == cfg.selectedPiece.Tile {
			startingPosition = [2]int{rowIdx, i}
			break
		}
	}

	var pieceColor string

	if cfg.selectedPiece.IsWhite {
		pieceColor = "white"
	} else {
		pieceColor = "black"
	}

	if cfg.selectedPiece.IsPawn {
		gcfg.getPawnMoves(&possibleMoves, startingPosition, cfg.selectedPiece, currentGame)
	} else {
		for _, move := range cfg.selectedPiece.LegalMoves {
			gcfg.getMoves(&possibleMoves, startingPosition, move, cfg.selectedPiece.MovesOnce, pieceColor, currentGame)
		}
	}

	return possibleMoves
}

// TODO: IMPLEMENT EN PESSANT
func (gcfg *gameConfig) getPawnMoves(possible *[]string, startingPosition [2]int, piece components.Piece, currentGame string) {
	cfg := gcfg.Matches[currentGame]
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
		pieceOnCurrentTile := cfg.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		}
	}

	if startingPosition[1]-1 >= 0 {
		currentTile := mockBoard[currentPosition[0]][startingPosition[1]-1]
		pieceOnCurrentTile := cfg.board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		}
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := cfg.board[currentTile].Piece

	if pieceOnCurrentTile.Name != "" {
		return
	}

	*possible = append(*possible, currentTile)

	if !piece.Moved {
		tile := mockBoard[currentPosition[0]+moveIndex][currentPosition[1]]
		pT := cfg.board[tile].Piece
		if pT.Name == "" {
			*possible = append(*possible, tile)
		}
	}
}

func (gcfg *gameConfig) getMoves(possible *[]string, startingPosition [2]int, move []int, checkOnce bool, pieceColor, currentGame string) {
	cfg := gcfg.Matches[currentGame]
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := cfg.board[currentTile].Piece

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

	gcfg.getMoves(possible, currentPosition, move, checkOnce, pieceColor, currentGame)
}

func samePiece(selectedPiece, currentPiece components.Piece) bool {
	if selectedPiece.IsWhite && currentPiece.IsWhite {
		return true
	} else if !selectedPiece.IsWhite && !currentPiece.IsWhite {
		return true
	}

	return false
}

func (gcfg *gameConfig) checkForCastle(b map[string]components.Square, selectedPiece, currentPiece components.Piece, currentGame string) (bool, bool) {
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
			return gcfg.handleChecksWhenKingMoves(tile, currentGame)
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

func (gcfg *gameConfig) handleCastle(w http.ResponseWriter, currentPiece components.Piece, currentGame string) error {
	cfg := gcfg.Matches[currentGame]
	var king components.Piece
	var rook components.Piece

	if cfg.selectedPiece.IsKing {
		king = cfg.selectedPiece
		rook = currentPiece
	} else {
		king = currentPiece
		rook = cfg.selectedPiece
	}

	kTile := king.Tile
	rTile := rook.Tile
	savedKingTile := cfg.board[king.Tile]
	savedRookTile := cfg.board[rook.Tile]
	kingSquare := cfg.board[king.Tile]
	rookSquare := cfg.board[rook.Tile]

	if kingSquare.Coordinates[1] < rookSquare.Coordinates[1] {
		kC := kingSquare.Coordinates[1]
		rookSquare.Coordinates[1] = kC + cfg.coordinateMultiplier
		kingSquare.Coordinates[1] = kC + cfg.coordinateMultiplier*2
	} else {
		kC := kingSquare.Coordinates[1]
		rookSquare.Coordinates[1] = kC - cfg.coordinateMultiplier
		kingSquare.Coordinates[1] = kC - cfg.coordinateMultiplier*2
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

	if err != nil {
		return err
	}

	rowIdx := rowIdxMap[string(king.Tile[0])]
	king.Tile = mockBoard[rowIdx][kingSquare.Coordinates[1]/100]
	rook.Tile = mockBoard[rowIdx][rookSquare.Coordinates[1]/100]
	newKingSquare := cfg.board[king.Tile]
	newRookSquare := cfg.board[rook.Tile]
	newKingSquare.Piece = king
	newRookSquare.Piece = rook
	cfg.board[king.Tile] = newKingSquare
	cfg.board[rook.Tile] = newRookSquare
	cfg.pieces[king.Name] = king
	cfg.pieces[rook.Name] = rook
	savedKingTile.Piece = components.Piece{}
	savedRookTile.Piece = components.Piece{}
	cfg.board[kTile] = savedKingTile
	cfg.board[rTile] = savedRookTile
	cfg.selectedPiece = components.Piece{}
	cfg.isWhiteTurn = !cfg.isWhiteTurn
	gcfg.Matches[currentGame] = cfg
	go gcfg.gameDone(currentGame)

	return nil
}

func (gcfg *gameConfig) handleCheckForCheck(currentSquareName, currentGame string, selectedPiece components.Piece) (bool, components.Piece, []string) {
	cfg := gcfg.Matches[currentGame]
	var startingPosition [2]int

	var king components.Piece
	var pieceColor string

	savedStartingTile := selectedPiece.Tile
	savedStartSqua := cfg.board[savedStartingTile]
	saved := cfg.board[currentSquareName]

	if currentSquareName != "" {
		startingSquare := cfg.board[selectedPiece.Tile]
		startingSquare.Piece = components.Piece{}
		cfg.board[selectedPiece.Tile] = startingSquare
		selectedPiece.Tile = currentSquareName
		curSq := cfg.board[currentSquareName]
		curSq.Piece = selectedPiece
		cfg.board[currentSquareName] = curSq

		var kingName string
		if selectedPiece.IsWhite {
			kingName = "white_king"
			pieceColor = "white"
		} else {
			kingName = "black_king"
			pieceColor = "black"
		}

		king = cfg.pieces[kingName]
	} else {
		var kingName string
		if selectedPiece.IsWhite {
			kingName = "black_king"
			pieceColor = "black"
		} else {
			kingName = "white_king"
			pieceColor = "white"
		}
		king = cfg.pieces[kingName]
	}

	rowIdx := rowIdxMap[string(king.Tile[0])]

	for i := 0; i < len(mockBoard[rowIdx]); i++ {
		if mockBoard[rowIdx][i] == king.Tile {
			startingPosition = [2]int{rowIdx, i}
			break
		}
	}

	kingLegalMoves := [][]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}, {2, 1}, {2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}, {-2, 1}, {-2, -1}}

	// isPawn := strings.Contains(selectedPiece.Name, "pawn")
	// if isPawn {
	// cfg.getPawnMoves(&possibleMoves, startingPosition, selectedPiece)
	// } else {

	var tilesComb []string

	var check bool

	for _, move := range kingLegalMoves {
		var tilesUnderCheck []string
		checkInFor := gcfg.checkCheck(&tilesUnderCheck, startingPosition, startingPosition, move, pieceColor, currentGame)
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
		cfg.board[savedStartingTile] = savedStartSqua
		cfg.board[currentSquareName] = saved
		selectedPiece.Tile = savedStartingTile
		gcfg.Matches[currentGame] = cfg

		return check, king, tilesComb
	}

	return false, king, []string{}
}

func (gcfg *gameConfig) checkCheck(tilesUnderCheck *[]string, startingPosition, startPosCompare [2]int, move []int, pieceColor, currentGame string) bool {
	cfg := gcfg.Matches[currentGame]
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return false
	}

	if currentPosition[0] >= len(mockBoard) || currentPosition[1] >= len(mockBoard[startingPosition[0]]) {
		return false
	}

	currentTile := mockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := cfg.board[currentTile].Piece

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

	check := gcfg.checkCheck(tilesUnderCheck, currentPosition, startPosCompare, move, pieceColor, currentGame)
	if check {
		*tilesUnderCheck = append(*tilesUnderCheck, currentTile)
	}

	return check
}

func (gcfg *gameConfig) handleChecksWhenKingMoves(currentSquareName, currentGame string) bool {
	cfg := gcfg.Matches[currentGame]
	var kingPosition [2]int
	var king components.Piece
	var pieceColor string

	if cfg.isWhiteTurn {
		king = cfg.pieces["white_king"]
		pieceColor = "white"
	} else {
		king = cfg.pieces["black_king"]
		pieceColor = "black"
	}

	savedStartingTile := king.Tile
	savedStartSqua := cfg.board[savedStartingTile]
	saved := cfg.board[currentSquareName]

	startingSquare := cfg.board[king.Tile]
	startingSquare.Piece = components.Piece{}
	cfg.board[king.Tile] = startingSquare
	king.Tile = currentSquareName
	curSq := cfg.board[currentSquareName]
	curSq.Piece = king
	cfg.board[currentSquareName] = curSq

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
		if gcfg.checkCheck(&tilesUnderCheck, kingPosition, kingPosition, move, pieceColor, currentGame) {
			cfg.board[savedStartingTile] = savedStartSqua
			cfg.board[currentSquareName] = saved
			king.Tile = savedStartingTile
			return true
		}
	}

	cfg.board[savedStartingTile] = savedStartSqua
	cfg.board[currentSquareName] = saved
	king.Tile = savedStartingTile

	return false
}

func (gcfg *gameConfig) gameDone(currentGame string) {
	cfg := gcfg.Matches[currentGame]
	var king components.Piece
	if cfg.isWhiteTurn {
		king = cfg.pieces["white_king"]
	} else {
		king = cfg.pieces["black_king"]
	}

	savePiece := cfg.selectedPiece
	cfg.selectedPiece = king

	gcfg.Matches[currentGame] = cfg

	legalMoves := gcfg.checkLegalMoves(currentGame)

	cfg.selectedPiece = savePiece
	gcfg.Matches[currentGame] = cfg
	var checkCount []bool

	for _, move := range legalMoves {
		if gcfg.handleChecksWhenKingMoves(move, currentGame) {
			checkCount = append(checkCount, true)
		}
	}

	if len(legalMoves) == len(checkCount) {
		if cfg.isWhiteTurn && cfg.isWhiteUnderCheck {
			for _, piece := range cfg.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := cfg.selectedPiece
					cfg.selectedPiece = piece
					gcfg.Matches[currentGame] = cfg
					legalMoves := gcfg.checkLegalMoves(currentGame)
					cfg.selectedPiece = savePiece
					gcfg.Matches[currentGame] = cfg

					for _, move := range legalMoves {
						if slices.Contains(cfg.tilesUnderAttack, move) {
							return
						}
					}
				}
			}
			fmt.Println("checkmate")
		} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck {
			for _, piece := range cfg.pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := cfg.selectedPiece
					cfg.selectedPiece = piece
					gcfg.Matches[currentGame] = cfg
					legalMoves := gcfg.checkLegalMoves(currentGame)
					cfg.selectedPiece = savePiece
					gcfg.Matches[currentGame] = cfg

					for _, move := range legalMoves {
						if slices.Contains(cfg.tilesUnderAttack, move) {
							return
						}
					}
				}
			}
			fmt.Println("checkmate")
		} else if cfg.isWhiteTurn {
			for _, piece := range cfg.pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := cfg.selectedPiece
					cfg.selectedPiece = piece
					gcfg.Matches[currentGame] = cfg
					legalMoves := gcfg.checkLegalMoves(currentGame)
					cfg.selectedPiece = savePiece
					gcfg.Matches[currentGame] = cfg

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			fmt.Println("stalemate")
		} else if !cfg.isWhiteTurn {
			for _, piece := range cfg.pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := cfg.selectedPiece
					cfg.selectedPiece = piece
					gcfg.Matches[currentGame] = cfg
					legalMoves := gcfg.checkLegalMoves(currentGame)
					cfg.selectedPiece = savePiece
					gcfg.Matches[currentGame] = cfg

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

func handleIfCheck(w http.ResponseWriter, gcfg *gameConfig, selected components.Piece, currentGame string) bool {
	cfg := gcfg.Matches[currentGame]
	check, king, tilesUnderAttack := gcfg.handleCheckForCheck("", currentGame, selected)
	kingSquare := cfg.board[king.Tile]

	if check {
		setUserCheck(king, &cfg)
		err := respondWithCheck(w, kingSquare, king)

		if err != nil {
			fmt.Println(err)
		}

		cfg.tilesUnderAttack = tilesUnderAttack
		gcfg.Matches[currentGame] = cfg

		for _, tile := range tilesUnderAttack {
			t := cfg.board[tile]

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

func bigCleanup(currentSquareName string, cfg *Match) {
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile
	selSeq := cfg.board[selectedSquare]
	currentSquare.Selected = false
	currentPiece := cfg.pieces[cfg.selectedPiece.Name]
	currentPiece.Tile = currentSquareName
	currentPiece.Moved = true
	cfg.pieces[cfg.selectedPiece.Name] = currentPiece
	currentSquare.Piece = currentPiece
	cfg.selectedPiece = components.Piece{}
	selSeq.Piece = cfg.selectedPiece
	cfg.board[selectedSquare] = selSeq
	cfg.board[currentSquareName] = currentSquare
}

func formatTime(seconds int) string {
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func (gcfg *gameConfig) endTurn(w http.ResponseWriter, r *http.Request, currentGame string) {
	cfg := gcfg.Matches[currentGame]
	if cfg.isWhiteTurn {
		cfg.whiteTimer += cfg.addition
	} else {
		cfg.blackTimer += cfg.addition
	}
	cfg.isWhiteTurn = !cfg.isWhiteTurn
	gcfg.Matches[currentGame] = cfg
	gcfg.gameDone(currentGame)
	gcfg.timerHandler(w, r)
}

func (cfg *apiConfig) refreshToken(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("refresh_token")

	if err != nil {
		fmt.Println(err)
		return
	}

	dbToken, err := cfg.database.SearchForToken(r.Context(), c.Value)

	if err != nil {
		fmt.Println(err)
		fmt.Println("jeste")
		return
	}

	if dbToken.ExpiresAt.Before(time.Now()) {
		fmt.Println("token expired")
		w.WriteHeader(http.StatusUnauthorized)
		newUser := CurrentUser{
			Name: "Guest",
		}
		cfg.user = newUser
		return
	}

	user, err := cfg.database.GetUserById(r.Context(), dbToken.UserID)

	if err != nil {
		fmt.Println("couldn't find user")
		w.WriteHeader(http.StatusInternalServerError)
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

func (cfg *apiConfig) checkUser(r *http.Request) {
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
			userDb, err := cfg.database.GetUserById(r.Context(), userId)

			if err != nil {
				fmt.Println(err)
				return
			}

			cfg.user.Id = userDb.ID
			cfg.user.Name = userDb.Name
			cfg.user.email = userDb.Email
		}
	}
}

func (cfg *apiConfig) middleWareCheckForUser(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	})
}
