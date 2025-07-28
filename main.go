package main

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

func main() {
	// component := hello("Nikola")
	component2 := headerTemplate("Nikola")

	http.Handle("/", templ.Handler(component2))

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
