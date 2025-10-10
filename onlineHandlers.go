package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
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
	disconnect := make(chan string)
	go func() {

		<-disconnect

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

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseNormalClosure {
				log.Println(err, "normal close")
				break
			}
			if strings.Contains(err.Error(), "websocket: RSV1 set, bad opcode 7, bad MASK") {
				disconnect <- "disconnected"
			}
			if err != nil {
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
	UpdateCoordinates(&match, whitePlayer.Multiplier)
	http.SetCookie(w, &startGame)

	_ = cfg.database.CreateMatchUser(r.Context(), database.CreateMatchUserParams{
		UserID:  whitePlayer.ID,
		MatchID: match.matchId,
	})

	err = layout.MainPageOnline(match.board, match.pieces, whitePlayer.Multiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack, true).Render(r.Context(), w)
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

	err1 := game["white"].Conn.WriteMessage(websocket.TextMessage, []byte("test"))
	err2 := game["black"].Conn.WriteMessage(websocket.TextMessage, []byte("test"))

	if err1 == nil && err2 == nil {
		_, err = fmt.Fprintf(w, `<div id="wait" hx-swap-oob="outerHTML"></div>`)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
			return
		}
	}

	var time int8
	var result string
	var winner string
	if userId == game["white"].ID {
		gamePlayer := game["black"]
		time = gamePlayer.ReconnectTimer
		time -= 1
		gamePlayer.ReconnectTimer = time
		game["black"] = gamePlayer
		result = "1-0"
		winner = "white"
	} else {
		gamePlayer := game["white"]
		time = gamePlayer.ReconnectTimer
		time -= 1
		gamePlayer.ReconnectTimer = time
		game["white"] = gamePlayer
		result = "0-1"
		winner = "black"
	}
	if time < 0 {
		_, err = fmt.Fprintf(w, `<div id="wait" hx-swap-oob="outerHTML"></div>`)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
			return
		}

		err := components.EndGameModal(result, winner).Render(r.Context(), w)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
			return
		}
		return
	}

	_, err = fmt.Fprintf(w, `<span id="waiting" hx-swap-oob="true">%v</span>`, time)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't send time", err)
		return
	}
}

func (cfg *appConfig) checkOnlineHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		return
	}

	if strings.Contains(c.Value, "online:") {
		err := components.ReconnectModal().Render(r.Context(), w)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "Couldn't render template", err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) cancelOnlineHandler(w http.ResponseWriter, r *http.Request) {
	currentGame, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}

	userToken, err := r.Cookie("access_token")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "user not found", err)
		return
	}

	userId, err := auth.ValidateJWT(userToken.Value, cfg.secret)
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "invalid token", err)
		return
	}

	saveGame := cfg.Matches[currentGame.Value]

	onlineGame := cfg.connections[currentGame.Value]

	var result string
	if onlineGame["white"].ID == userId {
		result = "0-1"
	} else {
		result = "1-0"
	}

	err = cfg.database.UpdateMatchOnEnd(r.Context(), database.UpdateMatchOnEndParams{
		Result: result,
		ID:     saveGame.matchId,
	})
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "error updating match", err)
		return
	}

	cGC := http.Cookie{
		Name:     "current_game",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cGC)

	_, err = fmt.Fprintf(w, `<div id="rec" hx-swap-oob="outerHTML"></div>`)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "error sending message", err)
		return
	}
}

func (cfg *appConfig) continueOnlineHandler(w http.ResponseWriter, r *http.Request) {
	currentGame, err := r.Cookie("current_game")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}

	userToken, err := r.Cookie("access_token")
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "user not found", err)
		return
	}

	userId, err := auth.ValidateJWT(userToken.Value, cfg.secret)
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "invalid token", err)
		return
	}

	user, err := cfg.database.GetUserById(r.Context(), userId)
	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "user not found", err)
		return
	}

	match, ok := cfg.Matches[currentGame.Value]

	onlineGame, ok2 := cfg.connections[currentGame.Value]
	if !ok || !ok2 {
		cGC := http.Cookie{
			Name:     "current_game",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, &cGC)
		match := cfg.Matches["initial"]
		cfg.fillBoard("initial")

		whitePlayer := components.PlayerStruct{
			Image:  "/assets/images/user-icon.png",
			Name:   user.Name,
			Timer:  formatTime(match.whiteTimer),
			Pieces: "white",
		}
		blackPlayer := components.PlayerStruct{
			Image:  "/assets/images/user-icon.png",
			Name:   "Opponent",
			Timer:  formatTime(match.blackTimer),
			Pieces: "black",
		}

		err = layout.MainPagePrivate(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack, true).Render(r.Context(), w)

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
			return
		}

		return
	}

	var blackPlayer components.OnlinePlayerStruct
	var whitePlayer components.OnlinePlayerStruct

	for color, player := range onlineGame {
		if color == "white" {
			whitePlayer = player
		}

		if color == "black" {
			blackPlayer = player
		}
	}

	var multiplier int

	if blackPlayer.ID == userId {
		multiplier = blackPlayer.Multiplier
	} else {
		multiplier = whitePlayer.Multiplier
	}

	err = layout.MainPageOnline(
		match.board,
		match.pieces,
		multiplier,
		whitePlayer,
		blackPlayer,
		match.takenPiecesWhite,
		match.takenPiecesBlack,
		true,
	).Render(r.Context(), w)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "websocket upgrade failed", err)
		return
	}
}

func (cfg *appConfig) cancelOnlineSearchHandler(w http.ResponseWriter, r *http.Request) {

	currentGame, err := r.Cookie("current_game")

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "failed getting the cookie", err)
		return
	}

	cGC := http.Cookie{
		Name:     "current_game",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cGC)

	delete(cfg.connections, currentGame.Value)

	_, err = w.Write([]byte{})
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "failed closing the modal", err)
		return
	}
}
