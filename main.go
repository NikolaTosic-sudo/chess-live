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
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		logError("Couldn't load env", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered. Error:\n", r)
		}
	}()

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
	gameRooms := make(map[string]OnlineGame, 0)

	cfg := appConfig{
		database:    dbQueries,
		secret:      secret,
		users:       make(map[uuid.UUID]User, 0),
		connections: gameRooms,
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
				allMoves:             []string{},
			},
		},
	}

	cur := cfg.Matches["initial"]
	UpdateCoordinates(&cur, cur.coordinateMultiplier)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", cfg.middleWareCheckForUser(cfg.boardHandler))
	http.HandleFunc("/private", cfg.middleWareCheckForUserPrivate(cfg.privateBoardHandler))
	http.HandleFunc("POST /start", cfg.startGameHandler)
	http.HandleFunc("POST /resume", cfg.resumeGameHandler)
	http.HandleFunc("POST /move", cfg.moveHandler)
	http.HandleFunc("POST /move-to", cfg.moveToHandler)
	http.HandleFunc("POST /cover-check", cfg.coverCheckHandler)
	http.HandleFunc("GET /timer", cfg.timerHandler)
	http.HandleFunc("GET /time-options", cfg.timeOptionHandler)
	http.HandleFunc("POST /set-time", cfg.setTimeOption)
	http.HandleFunc("POST /update-multiplier", cfg.updateMultiplerHandler)
	http.HandleFunc("GET /login", cfg.loginOpenHandler)
	http.HandleFunc("GET /logout", cfg.logoutHandler)
	http.HandleFunc("GET /close-modal", cfg.closeModalHandler)
	http.HandleFunc("GET /login-modal", cfg.loginModalHandler)
	http.HandleFunc("GET /signup-modal", cfg.signupModalHandler)
	http.HandleFunc("POST /auth-signup", cfg.signupHandler)
	http.HandleFunc("POST /auth-login", cfg.loginHandler)
	http.HandleFunc("GET /api/refresh", cfg.refreshToken)
	http.HandleFunc("GET /all-moves", cfg.getAllMovesHandler)
	http.HandleFunc("GET /match-history", cfg.middleWareCheckForUserPrivate(cfg.matchHistoryHandler))
	http.HandleFunc("GET /play-game", cfg.playHandler)
	http.HandleFunc("GET /matches/{id}", cfg.matchesHandler)
	http.HandleFunc("GET /move-history/{tile}", cfg.moveHistoryHandler)
	http.HandleFunc("POST /promotion", cfg.handlePromotion)
	http.HandleFunc("/online", cfg.wsHandler)
	http.HandleFunc("/play-online", cfg.onlineBoardHandler)
	http.HandleFunc("/searching", cfg.searchingOppHandler)
	http.HandleFunc("/end-game", cfg.endGameHandler)
	http.HandleFunc("/surrender", cfg.surrenderHandler)
	http.HandleFunc("/wait-reconnect", cfg.waitingForReconnect)
	http.HandleFunc("/check-online", cfg.checkOnlineHandler)
	http.HandleFunc("/cancel-online", cfg.cancelOnlineHandler)
	http.HandleFunc("/continue-online", cfg.continueOnlineHandler)
	http.HandleFunc("/handle-end", cfg.endModalHandler)
	http.HandleFunc("/cancel-online-search", cfg.cancelOnlineSearchHandler)

	fmt.Printf("Listening on :%v\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		logError("couldn't start the server", err)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
