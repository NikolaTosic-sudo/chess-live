package matches

import (
	"fmt"
	"strings"
)

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
