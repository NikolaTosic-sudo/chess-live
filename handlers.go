package main

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
)

func (cfg *apiConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fillBoard()
	err := board.GridBoard(cfg.board, cfg.pieces).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *apiConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentPieceName := r.Header.Get("Hx-Trigger")
	canPlay := cfg.canPlay(currentPieceName)
	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	currentPiece := cfg.pieces[currentPieceName]
	currentSquareName := currentPiece.Tile
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile

	legalMoves := cfg.checkLegalMoves()

	if canEat(cfg.selectedPiece.Name, currentPieceName) && slices.Contains(legalMoves, currentSquareName) {
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			cfg.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			cfg.selectedPiece.Image,
		)
		delete(cfg.pieces, currentPieceName)
		cfg.selectedPiece.Tile = currentSquareName
		cfg.pieces[cfg.selectedPiece.Name] = cfg.selectedPiece
		currentSquare.Piece = cfg.selectedPiece
		cfg.board[currentSquareName] = currentSquare
		cfg.selectedPiece = board.Piece{}
		cfg.isWhiteTurn = !cfg.isWhiteTurn
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		selSq := cfg.board[selectedSquare]
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			currentPieceName,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			currentPiece.Image,
			cfg.selectedPiece.Name,
			selSq.Coordinates[0],
			selSq.Coordinates[1],
			cfg.selectedPiece.Image,
		)
		cfg.selectedPiece = currentPiece
		return
	}

	if currentSquare.Selected {
		currentSquare.Selected = false
		cfg.selectedPiece = board.Piece{}
		cfg.board[currentSquareName] = currentSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`, currentPieceName, currentSquare.Coordinates[0], currentSquare.Coordinates[1], currentPiece.Image)
		return
	} else {
		currentSquare.Selected = true
		cfg.selectedPiece = currentPiece
		cfg.board[currentSquareName] = currentSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
			</span>
		`, currentPieceName, currentSquare.Coordinates[0], currentSquare.Coordinates[1], currentPiece.Image)
		return
	}
}

func (cfg *apiConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile
	selSeq := cfg.board[selectedSquare]

	legalMoves := cfg.checkLegalMoves()

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		fmt.Fprintf(w, `
			<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			currentSquareName,
			currentSquare.Color,
			cfg.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			cfg.selectedPiece.Image,
		)
		currentSquare.Selected = false
		currentPiece := cfg.pieces[cfg.selectedPiece.Name]
		currentPiece.Tile = currentSquareName
		cfg.pieces[cfg.selectedPiece.Name] = currentPiece
		currentSquare.Piece = currentPiece
		cfg.selectedPiece = board.Piece{}
		selSeq.Piece = cfg.selectedPiece
		cfg.board[selectedSquare] = selSeq
		cfg.board[currentSquareName] = currentSquare
		cfg.isWhiteTurn = !cfg.isWhiteTurn
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
