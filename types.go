package main

import "github.com/NikolaTosic-sudo/chess-live/containers/components"

type apiConfig struct {
	port                 string
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
}
