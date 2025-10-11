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
	connections map[string]OnlineGame
}

type User struct {
	Id    uuid.UUID
	Name  string
	Email string
}

type OnlineGame struct {
	players   map[string]components.OnlinePlayerStruct
	message   chan (string)
	playerMsg chan ([2]string)
}

type Match struct {
	board                 map[string]components.Square
	pieces                map[string]components.Piece
	selectedPiece         components.Piece
	coordinateMultiplier  int
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
