package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/NikolaTosic-sudo/chess-live/internal/matches"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		responses.LogError("Couldn't load env", err)
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
	startingBoard := matches.MakeBoard()
	startingPieces := matches.MakePieces()

	matches := matches.Matches{
		Matches: map[string]matches.Match{
			"initial": {
				Board:                startingBoard,
				Pieces:               startingPieces,
				SelectedPiece:        components.Piece{},
				CoordinateMultiplier: 80,
				IsWhiteTurn:          true,
				IsWhiteUnderCheck:    false,
				IsBlackUnderCheck:    false,
				WhiteTimer:           600,
				BlackTimer:           600,
				Addition:             0,
				AllMoves:             []string{},
			},
		},
	}

	cfg := appConfig{
		database: dbQueries,
		secret:   secret,
		users:    make(map[uuid.UUID]User, 0),
		Matches:  matches,
	}

	cur, _ := cfg.Matches.GetMatch("initial")
	cur.UpdateCoordinates(cur.CoordinateMultiplier)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	cfg.registerAllHandlers()

	err = http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		responses.LogError("couldn't start the server", err)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
