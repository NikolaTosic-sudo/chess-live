package main

import (
	"sync"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
)

type appConfig struct {
	mu          sync.RWMutex
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
	board                map[string]components.Square
	pieces               map[string]components.Piece
	selectedPiece        components.Piece
	result               chan string
	coordinateMultiplier int
	isWhiteTurn          bool
	isWhiteUnderCheck    bool
	isBlackUnderCheck    bool
	tilesUnderAttack     []string
	blackTimer           int
	whiteTimer           int
	addition             int
	allMoves             []string
	matchId              int32
	takenPiecesWhite     []string
	takenPiecesBlack     []string
}
