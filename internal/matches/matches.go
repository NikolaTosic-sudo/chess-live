package matches

import (
	"net/http"
	"slices"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
)

var MockBoard = [][]string{
	{"8a", "8b", "8c", "8d", "8e", "8f", "8g", "8h"},
	{"7a", "7b", "7c", "7d", "7e", "7f", "7g", "7h"},
	{"6a", "6b", "6c", "6d", "6e", "6f", "6g", "6h"},
	{"5a", "5b", "5c", "5d", "5e", "5f", "5g", "5h"},
	{"4a", "4b", "4c", "4d", "4e", "4f", "4g", "4h"},
	{"3a", "3b", "3c", "3d", "3e", "3f", "3g", "3h"},
	{"2a", "2b", "2c", "2d", "2e", "2f", "2g", "2h"},
	{"1a", "1b", "1c", "1d", "1e", "1f", "1g", "1h"},
}

var RowIdxMap = map[string]int{
	"8": 0,
	"7": 1,
	"6": 2,
	"5": 3,
	"4": 4,
	"3": 5,
	"2": 6,
	"1": 7,
}

func (m *Match) CheckForCastle(currentPiece components.Piece) (bool, bool) {
	selectedPiece := m.SelectedPiece
	b := m.Board
	if (selectedPiece.IsKing &&
		strings.Contains(currentPiece.Name, "rook") ||
		(strings.Contains(selectedPiece.Name, "rook") &&
			currentPiece.IsKing)) &&
		!selectedPiece.Moved &&
		!currentPiece.Moved {

		var selectedStartingPosition [2]int
		var currentStartingPosition [2]int
		var tilesForCastle []string

		rowIdx := RowIdxMap[string(selectedPiece.Tile[0])]

		for i := 0; i < len(MockBoard[rowIdx]); i++ {
			if MockBoard[rowIdx][i] == selectedPiece.Tile {
				selectedStartingPosition = [2]int{rowIdx, i}
			}
			if MockBoard[rowIdx][i] == currentPiece.Tile {
				currentStartingPosition = [2]int{rowIdx, i}
			}
		}

		if selectedStartingPosition[1] > currentStartingPosition[1] {
			for i := range selectedStartingPosition[1] - currentStartingPosition[1] - 1 {
				getSquare := MockBoard[selectedStartingPosition[0]][selectedStartingPosition[1]-i-1]
				tilesForCastle = append(tilesForCastle, getSquare)
				pieceOnSquare := b[getSquare]
				if pieceOnSquare.Piece.Name != "" {
					return false, false
				}
			}
		}
		if selectedStartingPosition[1] < currentStartingPosition[1] {
			for i := range currentStartingPosition[1] - selectedStartingPosition[1] - 1 {
				getSquare := MockBoard[selectedStartingPosition[0]][currentStartingPosition[1]-i-1]
				tilesForCastle = append(tilesForCastle, getSquare)
				pieceOnSquare := b[getSquare]
				if pieceOnSquare.Piece.Name != "" {
					return false, false
				}
			}
		}

		var kingCheck bool
		if slices.ContainsFunc(tilesForCastle, func(tile string) bool {
			return m.HandleChecksWhenKingMoves(tile)
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

func (m *Match) HandleCheckForCheck(currentSquareName string, selectedPiece components.Piece) (bool, components.Piece, []string) {
	var startingPosition [2]int

	var king components.Piece
	var pieceColor string

	savedStartingTile := selectedPiece.Tile
	savedStartSqua := m.Board[savedStartingTile]
	saved := m.Board[currentSquareName]

	if currentSquareName != "" {
		startingSquare := m.Board[selectedPiece.Tile]
		startingSquare.Piece = components.Piece{}
		m.Board[selectedPiece.Tile] = startingSquare
		selectedPiece.Tile = currentSquareName
		curSq := m.Board[currentSquareName]
		curSq.Piece = selectedPiece
		m.Board[currentSquareName] = curSq

		var kingName string
		if selectedPiece.IsWhite {
			kingName = "white_king"
			pieceColor = "white"
		} else {
			kingName = "black_king"
			pieceColor = "black"
		}

		king = m.Pieces[kingName]
	} else {
		var kingName string
		if selectedPiece.IsWhite {
			kingName = "black_king"
			pieceColor = "black"
		} else {
			kingName = "white_king"
			pieceColor = "white"
		}
		king = m.Pieces[kingName]
	}

	rowIdx := RowIdxMap[string(king.Tile[0])]

	for i := 0; i < len(MockBoard[rowIdx]); i++ {
		if MockBoard[rowIdx][i] == king.Tile {
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
		m.Board[savedStartingTile] = savedStartSqua
		m.Board[currentSquareName] = saved
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

	if currentPosition[0] >= len(MockBoard) || currentPosition[1] >= len(MockBoard[startingPosition[0]]) {
		return false
	}

	currentTile := MockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := m.Board[currentTile].Piece

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

func (m *Match) HandleChecksWhenKingMoves(currentSquareName string) bool {
	var kingPosition [2]int
	var king components.Piece
	var pieceColor string

	if m.IsWhiteTurn {
		king = m.Pieces["white_king"]
		pieceColor = "white"
	} else {
		king = m.Pieces["black_king"]
		pieceColor = "black"
	}

	savedStartingTile := king.Tile
	savedStartSqua := m.Board[savedStartingTile]
	saved := m.Board[currentSquareName]

	startingSquare := m.Board[king.Tile]
	startingSquare.Piece = components.Piece{}
	m.Board[king.Tile] = startingSquare
	king.Tile = currentSquareName
	curSq := m.Board[currentSquareName]
	curSq.Piece = king
	m.Board[currentSquareName] = curSq

	rowIdx := RowIdxMap[string(king.Tile[0])]

	for i := 0; i < len(MockBoard[rowIdx]); i++ {
		if MockBoard[rowIdx][i] == king.Tile {
			kingPosition = [2]int{rowIdx, i}
			break
		}
	}

	kingLegalMoves := [][]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}, {2, 1}, {2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}, {-2, 1}, {-2, -1}}

	for _, move := range kingLegalMoves {
		var tilesUnderCheck []string
		if m.checkCheck(&tilesUnderCheck, kingPosition, kingPosition, move, pieceColor) {
			m.Board[savedStartingTile] = savedStartSqua
			m.Board[currentSquareName] = saved
			king.Tile = savedStartingTile
			return true
		}
	}

	m.Board[savedStartingTile] = savedStartSqua
	m.Board[currentSquareName] = saved
	king.Tile = savedStartingTile

	return false
}

func (m *Match) HandleIfCheck(w http.ResponseWriter, r *http.Request, selected components.Piece) (bool, error) {
	check, king, tilesUnderAttack := m.HandleCheckForCheck("", selected)
	kingSquare := m.Board[king.Tile]
	if check {
		m.SetUserCheck(king)
		err := m.RespondWithCheck(w, kingSquare, king)
		if err != nil {
			return false, err
		}
		m.TilesUnderAttack = tilesUnderAttack
		for _, tile := range tilesUnderAttack {
			t := m.Board[tile]

			if t.Piece.Name != "" {
				err := responses.RespondWithNewPiece(w, r, t)

				if err != nil {
					return false, err
				}
			} else {
				err := m.RespondWithCoverCheck(w, tile, t)
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return true, nil
}

func (m *Match) CleanFillBoard(pieces map[string]components.Piece) {
	m.Pieces = pieces
	for i, tile := range m.Board {
		tile.Piece = components.Piece{}
		m.Board[i] = tile
	}
	for _, v := range pieces {
		getTile := m.Board[v.Tile]
		getTile.Piece = v
		m.Board[v.Tile] = getTile
	}
}

func (m *Matches) GetMatch(key string) (Match, bool) {
	match, ok := m.Matches[key]
	return match, ok
}

func (m *Matches) SetMatch(key string, match Match) {
	m.Matches[key] = match
}

func (m *Matches) GetInitialMatch() Match {
	match := m.Matches["initial"]
	return match
}

func (m *Matches) GetAllOnlineMatches() map[string]Match {
	onlineMatches := make(map[string]Match)

	for name, match := range m.Matches {
		if match.IsOnline {
			onlineMatches[name] = match
		}
	}

	return onlineMatches
}

func (m *Match) IsOnlineMatch() (OnlineGame, bool) {
	return m.Online, m.IsOnline
}
