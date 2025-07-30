package main

import "strings"

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

func (cfg *apiConfig) canEat(selectedPiece, currentPiece string) bool {
	if (strings.Contains(selectedPiece, "white") &&
		strings.Contains(currentPiece, "black")) ||
		(strings.Contains(selectedPiece, "black") &&
			strings.Contains(currentPiece, "white")) {
		return true
	}

	return false
}
