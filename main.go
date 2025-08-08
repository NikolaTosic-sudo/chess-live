package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
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

	cfg := apiConfig{
		database: dbQueries,
		secret:   secret,
		users:    make(map[uuid.UUID]CurrentUser, 0),
	}

	gcfg := gameConfig{
		Matches: map[string]Match{
			"initial": {
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
			},
		},
		secret: secret,
	}

	cur := gcfg.Matches["initial"]
	UpdateCoordinates(&cur)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.middleWareCheckForUser(gcfg.boardHandler))
	http.HandleFunc("/private", cfg.middleWareCheckForUserPrivate(gcfg.privateBoardHandler))
	http.HandleFunc("POST /start", gcfg.startGameHandler)
	http.HandleFunc("POST /resume", gcfg.resumeGameHandler)
	http.HandleFunc("POST /move", gcfg.moveHandler)
	http.HandleFunc("POST /move-to", gcfg.moveToHandler)
	http.HandleFunc("POST /cover-check", gcfg.coverCheckHandler)
	http.HandleFunc("GET /timer", gcfg.timerHandler)
	http.HandleFunc("GET /time-options", cfg.timeOptionHandler)
	http.HandleFunc("POST /set-time", gcfg.setTimeOption)
	http.HandleFunc("POST /update-multiplier", gcfg.updateMultiplerHandler)
	http.HandleFunc("GET /login", cfg.loginOpenHandler)
	http.HandleFunc("GET /logout", cfg.logoutHandler)
	http.HandleFunc("GET /close-modal", cfg.closeModalHandler)
	http.HandleFunc("GET /login-modal", cfg.loginModalHandler)
	http.HandleFunc("GET /signup-modal", cfg.signupModalHandler)
	http.HandleFunc("POST /auth-signup", cfg.signupHandler)
	http.HandleFunc("POST /auth-login", cfg.loginHandler)
	http.HandleFunc("GET /api/refresh", cfg.refreshToken)

	fmt.Printf("Listening on :%v\n", port)
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}
