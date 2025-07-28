package main

import (
	"fmt"
	"net/http"

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
