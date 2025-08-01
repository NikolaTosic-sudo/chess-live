package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	cfg := apiConfig{
		port:              port,
		board:             startingBoard,
		pieces:            startingPieces,
		selectedPiece:     board.Piece{},
		isWhiteTurn:       true,
		isWhiteUnderCheck: false,
		isBlackUnderCheck: false,
	}

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.boardHandler)
	http.HandleFunc("POST /move", cfg.moveHandler)
	http.HandleFunc("POST /move-to", cfg.moveToHandler)
	http.HandleFunc("POST /cover-check", cfg.coverCheckHandler)

	fmt.Printf("Listening on :%v\n", cfg.port)
	http.ListenAndServe(fmt.Sprintf(":%v", cfg.port), nil)
}
