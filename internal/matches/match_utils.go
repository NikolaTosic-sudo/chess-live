package matches

import (
	"fmt"
	"strings"

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
	} else {
		m.PossibleEnPessant = ""
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

func checkForNotEnoughPieces(pieces map[string]components.Piece) bool {
	if len(pieces) > 4 {
		return false
	}

	if len(pieces) == 2 {
		return true
	}

	whiteCount := 0
	blackCount := 0

	for _, piece := range pieces {
		if strings.Contains(piece.Image, "pawn") {
			return false
		}

		if strings.Contains(piece.Image, "queen") {
			return false
		}

		if strings.Contains(piece.Image, "rook") {
			return false
		}

		if piece.IsWhite {
			whiteCount += 1
		} else {
			blackCount += 1
		}
	}

	if whiteCount == 3 || blackCount == 3 {
		return false
	}

	return true
}

func checkForRepeatingMoves(match *Match) bool {
	snapshots := match.PiecesSnapshot
	n := len(snapshots)

	if n < 3 {
		return false
	}

	latest := snapshots[n-1]

	count := 0
	for i := 0; i < n-1; i += 2 {
		if samePosition(latest, snapshots[i]) {
			count++
		}
	}

	if count >= 2 {
		return true
	}

	if n > 10 {
		match.PiecesSnapshot = snapshots[n-10:]
	}

	return false
}

func samePosition(a, b map[string]components.Piece) bool {
	if len(a) != len(b) {
		return false
	}
	for name, p := range a {
		q, ok := b[name]
		if !ok {
			return false
		}
		if p.Tile != q.Tile || p.IsWhite != q.IsWhite {
			return false
		}
	}
	return true
}
