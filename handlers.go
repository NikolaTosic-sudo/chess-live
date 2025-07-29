package main

import (
	"fmt"
	"net/http"
	"strings"

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
		selSq := cfg.board[cfg.selectedSquare]
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			currentSquare,
			getSquare.Coordinates[0],
			getSquare.Coordinates[1],
			getSquare.Piece,
			cfg.selectedSquare,
			selSq.Coordinates[0],
			selSq.Coordinates[1],
			selSq.Piece,
		)
		cfg.selectedSquare = currentSquare
		return
	}

	if getSquare.Selected {
		getSquare.Selected = false
		cfg.selectedSquare = ""
		cfg.board[currentSquare] = getSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`, currentSquare, getSquare.Coordinates[0], getSquare.Coordinates[1], getSquare.Piece)
		return
	} else {
		getSquare.Selected = true
		cfg.selectedSquare = currentSquare
		cfg.board[currentSquare] = getSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
			</span>
		`, currentSquare, getSquare.Coordinates[0], getSquare.Coordinates[1], getSquare.Piece)
		return
	}
}

func (cfg *apiConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquare := r.Header.Get("Hx-Trigger")
	triggers := strings.Split(currentSquare, "-")
	getSquare := cfg.board[triggers[1]]

	if cfg.selectedSquare != "" && cfg.selectedSquare != triggers[1] {
		selSq := cfg.board[cfg.selectedSquare]
		fmt.Fprintf(w, `
			<div id="tile-%v" hx-post="/move-to" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			triggers[1],
			getSquare.Color,
			cfg.selectedSquare,
			getSquare.Coordinates[0],
			getSquare.Coordinates[1],
			selSq.Piece,
		)
		getSquare.Piece = selSq.Piece
		cfg.board[triggers[1]] = getSquare
		selSq.Piece = ""
		cfg.selectedSquare = ""
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
