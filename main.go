package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	dbUrl := os.Getenv("DB_URL")
	secret := os.Getenv("SECRET")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	user := CurrentUser{
		Name: "Guest",
	}

	cfg := apiConfig{
		database:             dbQueries,
		secret:               secret,
		user:                 user,
		board:                startingBoard,
		pieces:               startingPieces,
		selectedPiece:        components.Piece{},
		coordinateMultiplier: 80,
		isWhiteTurn:          true,
		isWhiteUnderCheck:    false,
		isBlackUnderCheck:    false,
		whiteTimer:           600,
		blackTimer:           600,
		addition:             0,
	}

	UpdateCoordinates(&cfg)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.boardHandler)
	http.HandleFunc("POST /start", cfg.startGameHandler)
	http.HandleFunc("POST /move", cfg.moveHandler)
	http.HandleFunc("POST /move-to", cfg.moveToHandler)
	http.HandleFunc("POST /cover-check", cfg.coverCheckHandler)
	http.HandleFunc("GET /timer", cfg.timerHandler)
	http.HandleFunc("GET /time-options", cfg.timeOptionHandler)
	http.HandleFunc("POST /set-time", cfg.setTimeOption)
	http.HandleFunc("POST /update-multiplier", cfg.updateMultiplerHandler)
	http.HandleFunc("GET /login", cfg.loginOpenHandler)
	http.HandleFunc("GET /close-modal", cfg.closeModalHandler)
	http.HandleFunc("GET /login-modal", cfg.loginModalHandler)
	http.HandleFunc("GET /signup-modal", cfg.signupModalHandler)
	http.HandleFunc("POST /auth-signup", cfg.signupHandler)
	http.HandleFunc("POST /auth-login", cfg.loginHandler)
	http.HandleFunc("GET /api/refresh", cfg.refreshToken)

	fmt.Printf("Listening on :%v\n", port)
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}
