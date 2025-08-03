package main

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/containers/errorPage"
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

func respondWithNewPiece(w http.ResponseWriter, square components.Square) error {
	_, err := fmt.Fprintf(w, `
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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

func respondWithCheck(w http.ResponseWriter, square components.Square, king components.Piece) error {
	_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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

func respondWithCoverCheck(w http.ResponseWriter, tile string, t components.Square) error {
	_, err := fmt.Fprintf(w, `
			<div id="%v" hx-post="/cover-check" hx-swap-oob="true" class="tile tile-md" style="background-color: %v"></div>
		`,
		tile,
		t.Color,
	)

	if err != nil {
		return err
	}

	return nil
}
