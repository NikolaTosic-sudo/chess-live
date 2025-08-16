package main

import (
	"log"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/gorilla/websocket"
)

func (cfg *appConfig) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	c, err := r.Cookie("current_game")
	if err != nil {
		log.Println("No game found:", err)
		return
	}
	userC, err := r.Cookie("access_token")
	if err != nil {
		log.Println("No user:", err)
		return
	}
	userId, err := auth.ValidateJWT(userC.Value, cfg.secret)
	game := cfg.connections[c.Value]
	for color, player := range game {
		if player.ID == userId {
			player.Conn = conn
			game[color] = player
		}
	}
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error from", err)
				break
			}

			for _, connect := range game {
				if conn != connect.Conn {
					err = connect.Conn.WriteMessage(websocket.TextMessage, msg)
					if err != nil {
						log.Println("write error to", err)
					}
				}
			}
		}
	}()
}

func (cfg *appConfig) searchingOppHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		log.Println("No game found:", err)
		return
	}
	currentGame := c.Value
	game := cfg.connections[currentGame]
	var emptyPlayer components.OnlinePlayerStruct
	if game["black"] == emptyPlayer {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	whitePlayer := game["white"]
	blackPlayer := game["black"]

	userCookie, err := r.Cookie("access_token")
	if err != nil {
		log.Println("No user found")
		return
	}

	userId, err := auth.ValidateJWT(userCookie.Value, cfg.secret)

	matchId, _ := cfg.database.CreateMatch(r.Context(), database.CreateMatchParams{
		White:    whitePlayer.Name,
		Black:    blackPlayer.Name,
		FullTime: 600,
		UserID:   userId,
		IsOnline: true,
	})

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	cfg.Matches[currentGame] = Match{
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
		matchId:              matchId,
	}

	startGame := http.Cookie{
		Name:     "current_game",
		Value:    currentGame,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	match := cfg.Matches[currentGame]
	cfg.fillBoard(currentGame)
	UpdateCoordinates(&match)
	http.SetCookie(w, &startGame)

	layout.MainPageOnline(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack, true).Render(r.Context(), w)
}
