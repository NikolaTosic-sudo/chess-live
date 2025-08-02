package main

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
	"github.com/NikolaTosic-sudo/chess-live/components/errorPage"
)

func respondWithAnError(w http.ResponseWriter, code int, message string, err error) {
	fmt.Printf("%v:%v\n", message, err)
	w.WriteHeader(code)
}

func respondWithAnErrorPage(w http.ResponseWriter, r *http.Request, code int, message string) {
	err := errorPage.Error(code, message).Render(r.Context(), w)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "Couldn't render template", err)
		return
	}
}

func respondWithNewPiece(w http.ResponseWriter, square board.Square) error {
	_, err := fmt.Fprintf(w, `
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
						<img src="/assets/pieces/%v.svg" />
					</span>
				`,
		square.Piece.Name,
		square.Coordinates[0],
		square.Coordinates[1],
		square.Piece.Image,
	)

	if err != nil {
		return err
	}

	return nil
}

func respondWithCheck(w http.ResponseWriter, square board.Square, king board.Piece) error {
	_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-red-400 " />
			</span>
		`,
		king.Name,
		square.Coordinates[0],
		square.Coordinates[1],
		king.Image,
	)

	if err != nil {
		return err
	}

	return nil
}

func respondWithCoverCheck(w http.ResponseWriter, tile string, t board.Square) error {
	_, err := fmt.Fprintf(w, `
			<div id="%v" hx-post="/cover-check" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>
		`,
		tile,
		t.Color,
	)

	if err != nil {
		return err
	}

	return nil
}
