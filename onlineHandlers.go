package main

import (
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
		for {
			_, msg, err := conn.ReadMessage()
			if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseNormalClosure {
				respondWithAnError(w, http.StatusTeapot, "websocket closed", err)
				break
			}
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't read message", err)
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
