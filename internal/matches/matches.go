package matches

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/NikolaTosic-sudo/chess-live/internal/utils"
	"github.com/google/uuid"
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

func (m *Match) CanPlay(piece components.Piece, onlineGame map[string]components.OnlinePlayerStruct, userId uuid.UUID) bool {
	if onlineGame != nil {

		if userId == uuid.Nil {
			return false
		}

		if piece.IsWhite && m.IsWhiteTurn && onlineGame["white"].ID == userId {
			return true
		} else if m.SelectedPiece.IsWhite && piece.IsWhite && m.IsWhiteTurn && onlineGame["white"].ID == userId {
			return true
		} else if !piece.IsWhite && !m.IsWhiteTurn && onlineGame["black"].ID == userId {
			return true
		} else if !m.SelectedPiece.IsWhite && !piece.IsWhite && !m.IsWhiteTurn && onlineGame["black"].ID == userId {
			return true
		}

		return false
	}
	if m.IsWhiteTurn {
		if piece.IsWhite {
			return true
		} else if m.SelectedPiece.IsWhite && piece.IsWhite {
			return true
		}
	} else {
		if !piece.IsWhite {
			return true
		} else if !m.SelectedPiece.IsWhite && !piece.IsWhite {
			return true
		}
	}

	return false
}

func (m *Match) FillBoard() {
	for _, v := range m.Pieces {
		getTile := m.Board[v.Tile]
		getTile.Piece = v
		m.Board[v.Tile] = getTile
	}
}

func (m *Match) CheckLegalMoves() []string {
	var startingPosition [2]int

	var possibleMoves []string

	if m.SelectedPiece.Tile == "" {
		return possibleMoves
	}

	rowIdx := RowIdxMap[string(m.SelectedPiece.Tile[0])]

	for i := 0; i < len(MockBoard[rowIdx]); i++ {
		if MockBoard[rowIdx][i] == m.SelectedPiece.Tile {
			startingPosition = [2]int{rowIdx, i}
			break
		}
	}

	var pieceColor string

	if m.SelectedPiece.IsWhite {
		pieceColor = "white"
	} else {
		pieceColor = "black"
	}

	if m.SelectedPiece.IsPawn {
		m.getPawnMoves(&possibleMoves, startingPosition)
	} else {
		for _, move := range m.SelectedPiece.LegalMoves {
			m.getMoves(&possibleMoves, startingPosition, move, m.SelectedPiece.MovesOnce, pieceColor)
		}
	}

	return possibleMoves
}

func (m *Match) getPawnMoves(possible *[]string, startingPosition [2]int) {
	var moveIndex int
	piece := m.SelectedPiece
	if piece.IsWhite {
		moveIndex = -1
	} else {
		moveIndex = 1
	}
	currentPosition := [2]int{startingPosition[0] + moveIndex, startingPosition[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return
	}

	if currentPosition[0] >= len(MockBoard) || currentPosition[1] >= len(MockBoard[startingPosition[0]]) {
		return
	}

	if startingPosition[1]+1 < len(MockBoard[startingPosition[0]]) {
		currentTile := MockBoard[currentPosition[0]][startingPosition[1]+1]
		pieceOnCurrentTile := m.Board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		} else if strings.Contains(m.PossibleEnPessant, currentTile) {
			*possible = append(*possible, fmt.Sprintf("enpessant_%v", currentTile))
		}
	}

	if startingPosition[1]-1 >= 0 {
		currentTile := MockBoard[currentPosition[0]][startingPosition[1]-1]
		pieceOnCurrentTile := m.Board[currentTile].Piece
		if pieceOnCurrentTile.Name != "" {
			*possible = append(*possible, currentTile)
		} else if strings.Contains(m.PossibleEnPessant, currentTile) {
			*possible = append(*possible, fmt.Sprintf("enpessant_%v", currentTile))
		}
	}

	currentTile := MockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := m.Board[currentTile].Piece

	if pieceOnCurrentTile.Name != "" {
		return
	}

	*possible = append(*possible, currentTile)

	if !piece.Moved {
		tile := MockBoard[currentPosition[0]+moveIndex][currentPosition[1]]
		pT := m.Board[tile].Piece
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

	if currentPosition[0] >= len(MockBoard) || currentPosition[1] >= len(MockBoard[startingPosition[0]]) {
		return
	}

	currentTile := MockBoard[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := m.Board[currentTile].Piece

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

func (m *Match) GameDone(w http.ResponseWriter) {
	var king components.Piece
	if m.IsWhiteTurn {
		king = m.Pieces["white_king"]
	} else {
		king = m.Pieces["black_king"]
	}

	if m.MovesSinceLastCapture == 50 {
		msg, err := utils.TemplString(components.EndGameModal("1-1", ""))
		if err != nil {
			responses.LogError("couldn't render end game modal", err)
			return
		}
		err = m.SendMessage(w, msg, [2][]int{})
		if err != nil {
			responses.LogError("couldn't render end game modal", err)
			return
		}
		return
	}

	savePiece := m.SelectedPiece
	m.SelectedPiece = king
	legalMoves := m.CheckLegalMoves()
	m.SelectedPiece = savePiece
	var checkCount []bool
	for _, move := range legalMoves {
		if m.HandleChecksWhenKingMoves(move) {
			checkCount = append(checkCount, true)
		}
	}
	if len(legalMoves) == len(checkCount) {
		if m.IsWhiteTurn && m.IsWhiteUnderCheck {
			for _, piece := range m.Pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := m.SelectedPiece
					m.SelectedPiece = piece
					legalMoves := m.CheckLegalMoves()
					m.SelectedPiece = savePiece

					for _, move := range legalMoves {
						if slices.Contains(m.TilesUnderAttack, move) {
							return
						}
					}
				}
			}
			msg, err := utils.TemplString(components.EndGameModal("0-1", "black"))
			if err != nil {
				responses.LogError("couldn't convert component to string", err)
				return
			}
			err = m.SendMessage(w, msg, [2][]int{})
			if err != nil {
				responses.LogError("couldn't render end game modal", err)
				return
			}
		} else if !m.IsWhiteTurn && m.IsBlackUnderCheck {
			for _, piece := range m.Pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := m.SelectedPiece
					m.SelectedPiece = piece
					legalMoves := m.CheckLegalMoves()
					m.SelectedPiece = savePiece

					for _, move := range legalMoves {
						if slices.Contains(m.TilesUnderAttack, move) {
							return
						}
					}
				}
			}
			msg, err := utils.TemplString(components.EndGameModal("1-0", "white"))
			if err != nil {
				responses.LogError("couldn't convert component to string", err)
				return
			}
			err = m.SendMessage(w, msg, [2][]int{})
			if err != nil {
				responses.LogError("couldn't render end game modal", err)
				return
			}
		} else if m.IsWhiteTurn {
			for _, piece := range m.Pieces {
				if piece.IsWhite && !piece.IsKing {
					savePiece := m.SelectedPiece
					m.SelectedPiece = piece
					legalMoves := m.CheckLegalMoves()
					m.SelectedPiece = savePiece

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			msg, err := utils.TemplString(components.EndGameModal("1-1", ""))
			if err != nil {
				responses.LogError("couldn't convert component to string", err)
				return
			}
			err = m.SendMessage(w, msg, [2][]int{})
			if err != nil {
				responses.LogError("couldn't render end game modal", err)
				return
			}
		} else if !m.IsWhiteTurn {
			for _, piece := range m.Pieces {
				if !piece.IsWhite && !piece.IsKing {
					savePiece := m.SelectedPiece
					m.SelectedPiece = piece
					legalMoves := m.CheckLegalMoves()
					m.SelectedPiece = savePiece

					if len(legalMoves) > 0 {
						return
					}
				}
			}
			msg, err := utils.TemplString(components.EndGameModal("1-1", ""))
			if err != nil {
				responses.LogError("couldn't convert component to string", err)
				return
			}
			err = m.SendMessage(w, msg, [2][]int{})
			if err != nil {
				responses.LogError("couldn't render end game modal", err)
				return
			}
		}
	}
}

func (m *Match) SetUserCheck(king components.Piece) {
	if king.IsWhite {
		m.IsWhiteUnderCheck = true
	} else {
		m.IsBlackUnderCheck = true
	}
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

func (m *Match) BigCleanup(currentSquareName string) {
	currentSquare := m.Board[currentSquareName]
	selectedSquare := m.SelectedPiece.Tile
	selSeq := m.Board[selectedSquare]
	currentSquare.Selected = false
	currentPiece := m.Pieces[m.SelectedPiece.Name]
	currentPiece.Tile = currentSquareName
	currentPiece.Moved = true
	m.Pieces[m.SelectedPiece.Name] = currentPiece
	currentSquare.Piece = currentPiece
	m.SelectedPiece = components.Piece{}
	selSeq.Piece = m.SelectedPiece
	m.Board[selectedSquare] = selSeq
	m.Board[currentSquareName] = currentSquare
}

func (m *Match) EndTurn(w http.ResponseWriter) {
	if m.IsWhiteTurn {
		m.WhiteTimer += m.Addition
	} else {
		m.BlackTimer += m.Addition
	}
	m.IsWhiteTurn = !m.IsWhiteTurn
	m.GameDone(w)
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

func (m *Match) CheckForPawnPromotion(pawnName string, w http.ResponseWriter, userId uuid.UUID) (bool, error) {
	var isOnLastTile bool
	onlineGame, found := m.IsOnlineMatch()
	pawn := m.Pieces[pawnName]
	if !pawn.IsPawn {
		return false, nil
	}
	square := m.Board[pawn.Tile]
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
		for _, player := range onlineGame.Players {
			if player.ID == userId {
				multiplier = player.Multiplier
			}
		}
	} else {
		multiplier = m.CoordinateMultiplier
	}
	endBoardCoordinates := 7 * multiplier
	dropdownPosition := square.Coordinates[1] + multiplier
	if square.Coordinates[1] == endBoardCoordinates {
		dropdownPosition = square.Coordinates[1] - multiplier
	}

	rowIdx := RowIdxMap[string(pawn.Tile[0])]
	if pawn.IsWhite && rowIdx == 0 || !pawn.IsWhite && rowIdx == 7 {
		isOnLastTile = true
		message := fmt.Sprintf(
			responses.GetPromotionInitMessage(),
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

func (m *Match) CheckForEnPessant(selectedSquare string, currentSquare components.Square) {
	if m.SelectedPiece.IsPawn && !m.SelectedPiece.Moved {
		if m.Board[selectedSquare].CoordinatePosition[0]-currentSquare.CoordinatePosition[0] == 2 {
			freeTile := MockBoard[currentSquare.CoordinatePosition[0]-2][currentSquare.CoordinatePosition[1]]
			m.PossibleEnPessant = fmt.Sprintf("white_%v", freeTile)
		} else if currentSquare.CoordinatePosition[0]-m.Board[selectedSquare].CoordinatePosition[0] == 2 {
			freeTile := MockBoard[currentSquare.CoordinatePosition[0]+2][currentSquare.CoordinatePosition[1]]
			m.PossibleEnPessant = fmt.Sprintf("black_%v", freeTile)
		}
	}
}

func (m *Match) EatCleanup(pieceToDelete components.Piece, squareToDeleteName, currentSquareName string) (components.Square, components.Piece) {
	squareToDelete := m.Board[squareToDeleteName]
	currentSquare := m.Board[currentSquareName]

	m.AllMoves = append(m.AllMoves, currentSquareName)
	delete(m.Pieces, pieceToDelete.Name)
	m.SelectedPiece.Tile = currentSquareName
	m.Pieces[m.SelectedPiece.Name] = m.SelectedPiece
	currentSquare.Piece = m.SelectedPiece
	squareToDelete.Piece = components.Piece{}
	m.Board[currentSquareName] = currentSquare
	m.Board[squareToDeleteName] = squareToDelete
	saveSelected := m.SelectedPiece
	m.SelectedPiece = components.Piece{}
	m.PossibleEnPessant = ""
	m.MovesSinceLastCapture = 0

	return squareToDelete, saveSelected
}

func (m *Match) SendMessage(w http.ResponseWriter, msg string, args [2][]int) error {
	onlineGame, found := m.IsOnlineMatch()

	if found && len(args) > 0 {
		for _, onlinePlayer := range onlineGame.Players {
			var bottomCoordinates []int
			for _, coordinate := range args[0] {
				bottomCoordinates = append(bottomCoordinates, coordinate*onlinePlayer.Multiplier)
			}
			var leftCoordinates []int
			for _, coordinate := range args[1] {
				leftCoordinates = append(leftCoordinates, coordinate*onlinePlayer.Multiplier)
			}

			newMessage := utils.ReplaceStyles(msg, bottomCoordinates, leftCoordinates)
			onlineGame.PlayerMsg <- newMessage
			onlineGame.Player <- onlinePlayer
		}

	} else if found {
		onlineGame.Message <- msg
	} else {
		_, err := fmt.Fprint(w, msg)
		if err != nil {
			return err
		}
	}
	return nil
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

func (m *Match) RespondWithCheck(w http.ResponseWriter, square components.Square, king components.Piece) error {
	className := `class="bg-red-400"`
	message := fmt.Sprintf(
		responses.GetSinglePieceMessage(),
		king.Name,
		square.Coordinates[0],
		square.Coordinates[1],
		king.Image,
		className,
	)

	err := m.SendMessage(w, message, [2][]int{
		{square.CoordinatePosition[0]},
		{square.CoordinatePosition[1]},
	})

	return err
}

func (m *Match) RespondWithCoverCheck(w http.ResponseWriter, tile string, t components.Square) error {
	message := fmt.Sprintf(
		responses.GetTileMessage(),
		tile,
		"cover-check",
		t.Color,
	)

	err := m.SendMessage(w, message, [2][]int{})

	return err
}

func (m *Match) UpdateCoordinates(multiplier int) {
	for key, square := range m.Board {
		square.Coordinates[0] = square.CoordinatePosition[0] * multiplier
		square.Coordinates[1] = square.CoordinatePosition[1] * multiplier

		m.Board[key] = square
	}
}

func CanEat(selectedPiece, currentPiece components.Piece) bool {
	if (selectedPiece.IsWhite &&
		!currentPiece.IsWhite) ||
		(!selectedPiece.IsWhite &&
			currentPiece.IsWhite) {
		return true
	}

	return false
}

func SamePiece(selectedPiece, currentPiece components.Piece) bool {
	if selectedPiece.IsWhite && currentPiece.IsWhite {
		return true
	} else if !selectedPiece.IsWhite && !currentPiece.IsWhite {
		return true
	}

	return false
}
