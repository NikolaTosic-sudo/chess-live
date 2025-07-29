package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")

	startingBoard := MakeBoard()

	cfg := apiConfig{
		port:           port,
		board:          startingBoard,
		selectedSquare: "",
	}

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.boardHandler)
	http.HandleFunc("POST /move", cfg.moveHandler)
	http.HandleFunc("POST /move-to", cfg.moveToHandler)

	fmt.Printf("Listening on :%v\n", cfg.port)
	http.ListenAndServe(fmt.Sprintf(":%v", cfg.port), nil)
}
