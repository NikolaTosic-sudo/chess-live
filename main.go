package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")

	cfg := apiConfig{
		port: port,
	}

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.headerHandler)

	fmt.Printf("Listening on :%v\n", cfg.port)
	http.ListenAndServe(fmt.Sprintf(":%v", cfg.port), nil)
}
