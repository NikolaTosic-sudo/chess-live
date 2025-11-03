package matches

import (
	"fmt"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/google/uuid"
)

func (m *Match) FillBoard() {
	for _, v := range m.Pieces {
		getTile := m.Board[v.Tile]
		getTile.Piece = v
		m.Board[v.Tile] = getTile
	}
}

func (m *Match) SetUserCheck(king components.Piece) {
	if king.IsWhite {
		m.IsWhiteUnderCheck = true
	} else {
		m.IsBlackUnderCheck = true
	}
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
