package components

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var cols = [8]string{"a", "b", "c", "d", "e", "f", "g", "h"}
var rows = [8]int{8, 7, 6, 5, 4, 3, 2, 1}

type Square struct {
	Piece              Piece
	Selected           bool
	CoordinatePosition [2]int
	Coordinates        [2]int
	Color              string
}

type Piece struct {
	Name       string
	Image      string
	Tile       string
	IsWhite    bool
	LegalMoves [][]int
	MovesOnce  bool
	Moved      bool
	IsKing     bool
	IsPawn     bool
}

type PlayerStruct struct {
	Name   string
	Image  string
	Timer  string
	Pieces string
}

type OnlinePlayerStruct struct {
	ID             uuid.UUID
	Name           string
	Image          string
	Timer          string
	Pieces         string
	Conn           *websocket.Conn
	ReconnectTimer int8
}

type MatchStruct struct {
	White   string
	Black   string
	Ended   bool
	Date    string
	NoMoves int
	Result  string
	Online  bool
	MatchId int
}

func genCol(color string) string {
	return "background-color: " + color
}

func getPosX(row, multiplier int) string {
	return fmt.Sprintf("top: %vpx", row*multiplier)
}

func getPosY(row, multiplier int) string {
	return fmt.Sprintf("left: %vpx", row*multiplier-15)
}

func getPiecePos(cord [2]int) string {
	return fmt.Sprintf("bottom: %vpx; left: %vpx", cord[0], cord[1])
}
