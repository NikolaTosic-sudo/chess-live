package main

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type appConfig struct {
	database    *database.Queries
	secret      string
	users       map[uuid.UUID]CurrentUser
	Matches     map[string]Match
	connections map[string]*websocket.Conn
}

type CurrentUser struct {
	Id    uuid.UUID
	Name  string
	Email string
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
	allMoves             []string
	matchId              int32
	takenPiecesWhite     []string
	takenPiecesBlack     []string
}
