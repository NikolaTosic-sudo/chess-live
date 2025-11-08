package matches

import (
	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/queue"
)

type OnlineGame struct {
	Players      map[string]components.OnlinePlayerStruct
	Message      chan (string)
	PlayerMsg    chan (string)
	Player       chan (components.OnlinePlayerStruct)
	PlayersQueue queue.PlayersQueue
}

type Matches struct {
	Matches map[string]Match
}

type Match struct {
	Board                 map[string]components.Square
	Pieces                map[string]components.Piece
	SelectedPiece         components.Piece
	CoordinateMultiplier  int
	IsWhiteTurn           bool
	IsWhiteUnderCheck     bool
	IsBlackUnderCheck     bool
	TilesUnderAttack      []string
	BlackTimer            int
	WhiteTimer            int
	Addition              int
	AllMoves              []string
	PiecesSnapshot        []map[string]components.Piece
	MatchId               int32
	MovesSinceLastCapture int8
	PossibleEnPessant     string
	TakenPiecesWhite      []string
	TakenPiecesBlack      []string
	IsOnline              bool
	Online                OnlineGame
}
