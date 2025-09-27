package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/gorilla/websocket"
)

func (cfg *appConfig) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "websocket upgrade failed", err)
		return
	}
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}
	userC, err := r.Cookie("access_token")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "no user found", err)
		return
	}
	userId, err := auth.ValidateJWT(userC.Value, cfg.secret)
	if err != nil {
		respondWithAnError(w, http.StatusUnauthorized, "unauthorized user", err)
		return
	}
	game := cfg.connections[c.Value]
	for color, player := range game {
		if player.ID == userId {
			player.Conn = conn
			game[color] = player
		}
	}
	go func() {
		defer func() {
			err := conn.Close()

			if err != nil {
				log.Println(err, "connection closed")
				return
			}

			for _, connect := range game {
				if conn != connect.Conn {
					msg, err := TemplString(components.WaitForReconnectModal())
					if err != nil {
						log.Println(err, "error")
						return
					}
					err = connect.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
					if err != nil {
						log.Println(err, "error")
						return
					}
				}

			}

		}()
		for {
			_, msg, err := conn.ReadMessage()
			if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseNormalClosure {
				log.Println(err, "normal close")
				break
			}
			if err != nil {
				log.Println(err, "neki drugi error")
				break
			}

			for _, connect := range game {
				if conn != connect.Conn {
					err = connect.Conn.WriteMessage(websocket.TextMessage, msg)
					if err != nil {
						logError("websocket write error", err)
					}
				}
			}
		}
	}()
}

func (cfg *appConfig) searchingOppHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
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

	err = layout.MainPageOnline(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack, true).Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) waitingForReconnect(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "current game unavailable", err)
		return
	}
	userC, err := r.Cookie("access_token")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "access token unavailable", err)
		return
	}
	userId, err := auth.ValidateJWT(userC.Value, cfg.secret)
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "couldn't validate jwt", err)
		return
	}
	game := cfg.connections[c.Value]
	var time int8
	if userId == game["white"].ID {
		gamePlayer := game["white"]
		time = gamePlayer.ReconnectTimer
		time = time - 1
		gamePlayer.ReconnectTimer = time
		game["white"] = gamePlayer
	} else {
		gamePlayer := game["black"]
		time = gamePlayer.ReconnectTimer
		time = time - 1
		gamePlayer.ReconnectTimer = time
		game["black"] = gamePlayer
	}
	if time < 0 {
		log.Println("gotovo")
		return
	}

	_, err = fmt.Fprintf(w, `<span id="waiting" hx-swap-oob="true">%v</span>`, time)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't send time", err)
		return
	}
}
