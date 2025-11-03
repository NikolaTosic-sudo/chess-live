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
	//TODO: let's make board and pieces into it's own struct, so we can attach functions to it
	Board  map[string]components.Square
	Pieces map[string]components.Piece
	//TODO: from the top of my head there is a lot of stuff I do with selectedPiece, so maybe we can make this into it's own struct to attach methods? this one is for discussion
	SelectedPiece        components.Piece
	CoordinateMultiplier int
	IsWhiteTurn          bool
	IsWhiteUnderCheck    bool
	IsBlackUnderCheck    bool
	//TODO: same here as for selected piece, maybe even for allMoves, takenPiecesBlack and takenPiecesWhite
	TilesUnderAttack      []string
	BlackTimer            int
	WhiteTimer            int
	Addition              int
	AllMoves              []string
	MatchId               int32
	MovesSinceLastCapture int8
	PossibleEnPessant     string
	TakenPiecesWhite      []string
	TakenPiecesBlack      []string
	IsOnline              bool
	Online                OnlineGame
}
