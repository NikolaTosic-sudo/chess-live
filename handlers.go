package main

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
)

func (cfg *apiConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	// err := hello.HeaderTemplate("Nikola").Render(r.Context(), w)
	err := board.Board(cfg.board).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *apiConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentSquare := r.Header.Get("Hx-Trigger")
	getSquare := cfg.board[currentSquare]

	if getSquare.Selected {
		getSquare.Selected = false
		cfg.selectedSquare = ""
		cfg.board[currentSquare] = getSquare
		fmt.Println("vec je bio selektovan")
		return
	} else {
		getSquare.Selected = true
		cfg.selectedSquare = currentSquare
		cfg.board[currentSquare] = getSquare
		fmt.Println("nije bio selektovan")
		return
	}
}
