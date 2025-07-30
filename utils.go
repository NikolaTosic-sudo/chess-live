package main

import (
	"strings"
)

func (cfg *apiConfig) canPlay(pieceName string) bool {
	if cfg.isWhiteTurn {
		if strings.Contains(pieceName, "white") {
			return true
		} else if strings.Contains(cfg.selectedPiece.Name, "white") {
			return true
		}
	} else {
		if strings.Contains(pieceName, "black") {
			return true
		} else if strings.Contains(cfg.selectedPiece.Name, "black") {
			return true
		}
	}

	return false
}

func canEat(selectedPiece, currentPiece string) bool {
	if (strings.Contains(selectedPiece, "white") &&
		strings.Contains(currentPiece, "black")) ||
		(strings.Contains(selectedPiece, "black") &&
			strings.Contains(currentPiece, "white")) {
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
	mockBoard := [][]string{
		{"8a", "8b", "8c", "8d", "8e", "8f", "8g", "8h"},
		{"7a", "7b", "7c", "7d", "7e", "7f", "7g", "7h"},
		{"6a", "6b", "6c", "6d", "6e", "6f", "6g", "6h"},
		{"5a", "5b", "5c", "5d", "5e", "5f", "5g", "5h"},
		{"4a", "4b", "4c", "4d", "4e", "4f", "4g", "4h"},
		{"3a", "3b", "3c", "3d", "3e", "3f", "3g", "3h"},
		{"2a", "2b", "2c", "2d", "2e", "2f", "2g", "2h"},
		{"1a", "1b", "1c", "1d", "1e", "1f", "1g", "1h"},
	}

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

	if strings.Contains(cfg.selectedPiece.Name, "white") {
		pieceColor = "white"
	} else {
		pieceColor = "black"
	}

	for _, move := range cfg.selectedPiece.LegalMoves {
		cfg.getMoves(mockBoard, &possibleMoves, startingPosition, move, cfg.selectedPiece.MovesOnce, pieceColor)
	}

	return possibleMoves
}

func (cfg *apiConfig) getMoves(board [][]string, possible *[]string, startingPosition [2]int, move []int, checkOnce bool, pieceColor string) bool {
	currentPosition := [2]int{startingPosition[0] + move[0], startingPosition[1] + move[1]}

	if currentPosition[0] < 0 || currentPosition[1] < 0 {
		return false
	}

	if currentPosition[0] >= len(board) || currentPosition[1] >= len(board[startingPosition[0]]) {
		return false
	}

	currentTile := board[currentPosition[0]][currentPosition[1]]
	pieceOnCurrentTile := cfg.board[currentTile].Piece.Name

	if pieceOnCurrentTile != "" {
		if strings.Contains(pieceOnCurrentTile, pieceColor) {
			return false
		} else {
			*possible = append(*possible, currentTile)
			return false
		}
	}

	*possible = append(*possible, currentTile)

	if checkOnce {
		return false
	}

	cfg.getMoves(board, possible, currentPosition, move, checkOnce, pieceColor)

	return false
}
