package main

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
)

type appConfig struct {
	database *database.Queries
	secret   string
	users    map[uuid.UUID]User
	// TODO: this can probably be combined into one struct, instead of multiple, Matches and connections
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
	playerMsg chan (string)
	player    chan (components.OnlinePlayerStruct)
}

type Match struct {
	//TODO: let's make board and pieces into it's own struct, so we can attach functions to it
	board  map[string]components.Square
	pieces map[string]components.Piece
	//TODO: from the top of my head there is a lot of stuff I do with selectedPiece, so maybe we can make this into it's own struct to attach methods? this one is for discussion
	selectedPiece        components.Piece
	coordinateMultiplier int
	isWhiteTurn          bool
	isWhiteUnderCheck    bool
	isBlackUnderCheck    bool
	//TODO: same here as for selected piece, maybe even for allMoves, takenPiecesBlack and takenPiecesWhite
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
