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

	matches := Matches{
		matches: map[string]Match{
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

	cfg := appConfig{
		database:    dbQueries,
		secret:      secret,
		users:       make(map[uuid.UUID]User, 0),
		connections: gameRooms,
		Matches:     matches,
	}

	cur, _ := cfg.Matches.getMatch("initial")
	UpdateCoordinates(&cur, cur.coordinateMultiplier)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	cfg.registerAllHandlers()

	fmt.Printf("Listening on :%v\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		logError("couldn't start the server", err)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
