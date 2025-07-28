package main

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/components/hello"
)

func main() {

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hello.HeaderTemplate("Nikola").Render(r.Context(), w)
	})

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
