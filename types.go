package main

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	database *database.Queries
	secret   string
	user     CurrentUser
}

type CurrentUser struct {
	Id    uuid.UUID
	Name  string
	email string
}

type gameConfig struct {
	Matches map[string]Match
}

type Match struct {
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
