package matches

import (
	"net/http"
	"slices"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/NikolaTosic-sudo/chess-live/internal/utils"
)

func (m *Match) GameDone(w http.ResponseWriter) {
	var king components.Piece
	if m.IsWhiteTurn {
		king = m.Pieces["white_king"]
	} else {
		king = m.Pieces["black_king"]
	}

	if m.MovesSinceLastCapture == 50 {
		msg, err := utils.TemplString(components.EndGameModal("1-1", "", false))
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

	notEnoughPieces := checkForNotEnoughPieces(m.Pieces)

	if notEnoughPieces {
		msg, err := utils.TemplString(components.EndGameModal("1-1", "", false))
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
			msg, err := utils.TemplString(components.EndGameModal("0-1", "black", false))
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
			msg, err := utils.TemplString(components.EndGameModal("1-0", "white", false))
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
			msg, err := utils.TemplString(components.EndGameModal("1-1", "", false))
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
			msg, err := utils.TemplString(components.EndGameModal("1-1", "", false))
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
