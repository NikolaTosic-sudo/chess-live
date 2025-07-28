package main

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/components/hello"
)

func (cfg *apiConfig) headerHandler(w http.ResponseWriter, r *http.Request) {
	err := hello.HeaderTemplate("Nikola").Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}
