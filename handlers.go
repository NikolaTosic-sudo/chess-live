package main

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
)

func (cfg *apiConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	err := board.GridBoard(cfg.board).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *apiConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentSquare := r.Header.Get("Hx-Trigger")
	getSquare := cfg.board[currentSquare]

	if cfg.selectedSquare != "" && cfg.selectedSquare != currentSquare {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if getSquare.Selected {
		getSquare.Selected = false
		cfg.selectedSquare = ""
		cfg.board[currentSquare] = getSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`, currentSquare, getSquare.Coordinates[0], getSquare.Coordinates[1], getSquare.Piece)
		return
	} else {
		getSquare.Selected = true
		cfg.selectedSquare = currentSquare
		cfg.board[currentSquare] = getSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
			</span>
		`, currentSquare, getSquare.Coordinates[0], getSquare.Coordinates[1], getSquare.Piece)
		return
	}
}
