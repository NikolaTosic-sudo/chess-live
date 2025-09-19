package main

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
)

type appConfig struct {
	database    *database.Queries
	secret      string
	users       map[uuid.UUID]User
	Matches     map[string]Match
	connections map[string]map[string]components.OnlinePlayerStruct
}

type User struct {
	Id    uuid.UUID
	Name  string
	Email string
}

type Match struct {
	board                 map[string]components.Square
	pieces                map[string]components.Piece
	selectedPiece         components.Piece
	coordinateMultiplier  int
	disconnected          chan string
	isWhiteTurn           bool
	isWhiteUnderCheck     bool
	isBlackUnderCheck     bool
	tilesUnderAttack      []string
	blackTimer            int
	whiteTimer            int
	addition              int
	allMoves              []string
	matchId               int32
	movesSinceLastCapture int8
	possibleEnPessant     string
	takenPiecesWhite      []string
	takenPiecesBlack      []string
}
