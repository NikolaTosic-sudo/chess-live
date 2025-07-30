package main

import (
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
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

func (cfg *apiConfig) canPlay(piece board.Piece) bool {
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

func canEat(selectedPiece, currentPiece board.Piece) bool {
	if (selectedPiece.IsWhite &&
		!currentPiece.IsWhite) ||
		(!selectedPiece.IsWhite &&
			currentPiece.IsWhite) {
		return true
	}

	return false
}

func (cfg *apiConfig) fillBoard() {
	for _, v := range cfg.pieces {
		getTile := cfg.board[v.Tile]
		getTile.Piece = v
		cfg.board[v.Tile] = getTile
	}
}

func (cfg *apiConfig) checkLegalMoves() []string {
	rowIdxMap := map[string]int{
		"8": 0,
		"7": 1,
		"6": 2,
		"5": 3,
		"4": 4,
		"3": 5,
		"2": 6,
		"1": 7,
	}

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

	isPawn := strings.Contains(cfg.selectedPiece.Name, "pawn")

	if isPawn {
		cfg.getPawnMoves(&possibleMoves, startingPosition, cfg.selectedPiece)
	} else {
		for _, move := range cfg.selectedPiece.LegalMoves {
			cfg.getMoves(&possibleMoves, startingPosition, move, cfg.selectedPiece.MovesOnce, pieceColor)
		}
	}

	return possibleMoves
}

func (cfg *apiConfig) getPawnMoves(possible *[]string, startingPosition [2]int, piece board.Piece) {
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
		*possible = append(*possible, tile)
	}
}

func (cfg *apiConfig) getMoves(possible *[]string, startingPosition [2]int, move []int, checkOnce bool, pieceColor string) {
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

	cfg.getMoves(possible, currentPosition, move, checkOnce, pieceColor)
}

func samePiece(selectedPiece, currentPiece board.Piece) bool {
	if selectedPiece.IsWhite && currentPiece.IsWhite {
		return true
	} else if !selectedPiece.IsWhite && !currentPiece.IsWhite {
		return true
	}

	return false
}
