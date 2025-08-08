package main

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	database *database.Queries
	secret   string
	users    map[uuid.UUID]CurrentUser
}

type CurrentUser struct {
	Id    uuid.UUID
	Name  string
	Email string
}

type gameConfig struct {
	Matches map[string]Match
	secret  string
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
