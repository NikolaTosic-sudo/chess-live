package main

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
)

type apiConfig struct {
	database             *database.Queries
	board                map[string]components.Square
	pieces               map[string]components.Piece
	selectedPiece        components.Piece
	coordinateMultiplier int
	isWhiteTurn          bool
	isWhiteUnderCheck    bool
	isBlackUnderCheck    bool
	tilesUnderAttack     []string
	blackTimer           int
	whiteTimer           int
	addition             int
}
