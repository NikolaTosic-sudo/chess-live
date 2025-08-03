package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	cfg := apiConfig{
		port:                 port,
		board:                startingBoard,
		pieces:               startingPieces,
		selectedPiece:        components.Piece{},
		coordinateMultiplier: 80,
		isWhiteTurn:          true,
		isWhiteUnderCheck:    false,
		isBlackUnderCheck:    false,
		whiteTimer:           600,
		blackTimer:           600,
	}

	UpdateCoordinates(&cfg)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.boardHandler)
	http.HandleFunc("POST /move", cfg.moveHandler)
	http.HandleFunc("POST /move-to", cfg.moveToHandler)
	http.HandleFunc("POST /cover-check", cfg.coverCheckHandler)
	http.HandleFunc("GET /timer", cfg.timerHandler)
	http.HandleFunc("POST /update-multiplier", cfg.updateMultiplerHandler)

	fmt.Printf("Listening on :%v\n", cfg.port)
	http.ListenAndServe(fmt.Sprintf(":%v", cfg.port), nil)
}
