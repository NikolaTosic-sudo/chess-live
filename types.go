package main

import "github.com/NikolaTosic-sudo/chess-live/components/board"

type apiConfig struct {
	port           string
	board          map[string]board.Square
	selectedSquare string
}
