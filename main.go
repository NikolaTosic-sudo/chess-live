package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

	cfg := appConfig{
		database: dbQueries,
		secret:   secret,
		users:    make(map[uuid.UUID]CurrentUser, 0),
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
	UpdateCoordinates(&cur)

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
	http.HandleFunc("/ws", wsHandler)

	fmt.Printf("Listening on :%v\n", port)
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	counter := 0
	for {
		counter++
		message := fmt.Sprintf(`<div id="counter" class="text-white">Counter: %d</div>`, counter)
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
		time.Sleep(1 * time.Second)
	}
}
