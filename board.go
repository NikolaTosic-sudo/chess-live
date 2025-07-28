package main

import "github.com/NikolaTosic-sudo/chess-live/components/board"

func MakeBoard() map[string]board.Square {
	board := map[string]board.Square{
		"1a": {
			Piece:    "white_rook",
			Selected: false,
		},
		"2a": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3a": {
			Piece:    "",
			Selected: false,
		},
		"4a": {
			Piece:    "",
			Selected: false,
		},
		"5a": {
			Piece:    "",
			Selected: false,
		},
		"6a": {
			Piece:    "",
			Selected: false,
		},
		"7a": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8a": {
			Piece:    "black_rook",
			Selected: false,
		},
		"1b": {
			Piece:    "white_knight",
			Selected: false,
		},
		"2b": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3b": {
			Piece:    "",
			Selected: false,
		},
		"4b": {
			Piece:    "",
			Selected: false,
		},
		"5b": {
			Piece:    "",
			Selected: false,
		},
		"6b": {
			Piece:    "",
			Selected: false,
		},
		"7b": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8b": {
			Piece:    "black_knight",
			Selected: false,
		},
		"1c": {
			Piece:    "white_bishop",
			Selected: false,
		},
		"2c": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3c": {
			Piece:    "",
			Selected: false,
		},
		"4c": {
			Piece:    "",
			Selected: false,
		},
		"5c": {
			Piece:    "",
			Selected: false,
		},
		"6c": {
			Piece:    "",
			Selected: false,
		},
		"7c": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8c": {
			Piece:    "black_bishop",
			Selected: false,
		},
		"1d": {
			Piece:    "white_queen",
			Selected: false,
		},
		"2d": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3d": {
			Piece:    "",
			Selected: false,
		},
		"4d": {
			Piece:    "",
			Selected: false,
		},
		"5d": {
			Piece:    "",
			Selected: false,
		},
		"6d": {
			Piece:    "",
			Selected: false,
		},
		"7d": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8d": {
			Piece:    "black_queen",
			Selected: false,
		},
		"1e": {
			Piece:    "white_king",
			Selected: false,
		},
		"2e": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3e": {
			Piece:    "",
			Selected: false,
		},
		"4e": {
			Piece:    "",
			Selected: false,
		},
		"5e": {
			Piece:    "",
			Selected: false,
		},
		"6e": {
			Piece:    "",
			Selected: false,
		},
		"7e": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8e": {
			Piece:    "black_king",
			Selected: false,
		},
		"1f": {
			Piece:    "white_bishop",
			Selected: false,
		},
		"2f": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3f": {
			Piece:    "",
			Selected: false,
		},
		"4f": {
			Piece:    "",
			Selected: false,
		},
		"5f": {
			Piece:    "",
			Selected: false,
		},
		"6f": {
			Piece:    "",
			Selected: false,
		},
		"7f": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8f": {
			Piece:    "black_bishop",
			Selected: false,
		},
		"1g": {
			Piece:    "white_knight",
			Selected: false,
		},
		"2g": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3g": {
			Piece:    "",
			Selected: false,
		},
		"4g": {
			Piece:    "",
			Selected: false,
		},
		"5g": {
			Piece:    "",
			Selected: false,
		},
		"6g": {
			Piece:    "",
			Selected: false,
		},
		"7g": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8g": {
			Piece:    "black_knight",
			Selected: false,
		},
		"1h": {
			Piece:    "white_rook",
			Selected: false,
		},
		"2h": {
			Piece:    "white_pawn",
			Selected: false,
		},
		"3h": {
			Piece:    "",
			Selected: false,
		},
		"4h": {
			Piece:    "",
			Selected: false,
		},
		"5h": {
			Piece:    "",
			Selected: false,
		},
		"6h": {
			Piece:    "",
			Selected: false,
		},
		"7h": {
			Piece:    "black_pawn",
			Selected: false,
		},
		"8h": {
			Piece:    "black_rook",
			Selected: false,
		},
	}
	return board
}
