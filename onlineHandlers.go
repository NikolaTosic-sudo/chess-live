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
	"github.com/NikolaTosic-sudo/chess-live/internal/matches"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/NikolaTosic-sudo/chess-live/internal/utils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (cfg *appConfig) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "websocket upgrade failed", err)
		return
	}
	c, err := r.Cookie("current_game")
	if err != nil {
		return
	}

	userId, err := cfg.getUserId(r)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusUnauthorized, "unauthorized user", err)
		return
	}

	game := cfg.Matches.Matches[c.Value].Online
	for color, player := range game.Players {
		if player.ID == userId {
			player.Conn = conn
			game.Players[color] = player
		}
	}
	disconnect := make(chan string)

	go func() {
		for {
			select {
			case msg := <-game.Message:
				for playerColor, onlinePlayer := range game.Players {
					err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
					if err != nil {
						responses.RespondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
						break
					}
				}
			case playerMsg := <-game.PlayerMsg:
				player := <-game.Player
				err := player.Conn.WriteMessage(websocket.TextMessage, []byte(playerMsg))
				if err != nil {
					responses.RespondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerMsg), err)
					continue
				}
			}
		}
	}()

	go func() {

		<-disconnect

		err := conn.Close()

		if err != nil {
			log.Println(err, "connection closed")
			return
		}

		for _, connect := range game.Players {
			if conn != connect.Conn {
				msg, err := utils.TemplString(components.WaitForReconnectModal())
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
			_, _, err := conn.ReadMessage()
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
		}
	}()
}

func (cfg *appConfig) searchingOppHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}
	currentGame := c.Value

	game := cfg.Matches.Matches[currentGame].Online
	var emptyPlayer components.OnlinePlayerStruct
	if game.Players["black"] == emptyPlayer {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	whitePlayer := game.Players["white"]
	blackPlayer := game.Players["black"]
	startGame := cfg.makeCookie("current_game", currentGame, "/")

	match, _ := cfg.Matches.GetMatch(currentGame)
	match.FillBoard()
	match.UpdateCoordinates(whitePlayer.Multiplier)
	http.SetCookie(w, &startGame)

	_ = cfg.database.CreateMatchUser(r.Context(), database.CreateMatchUserParams{
		UserID:  whitePlayer.ID,
		MatchID: match.MatchId,
	})

	err = layout.MainPageOnline(match.Board, match.Pieces, whitePlayer.Multiplier, whitePlayer, blackPlayer, match.TakenPiecesWhite, match.TakenPiecesBlack, true).Render(r.Context(), w)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) waitingForReconnect(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "current game unavailable", err)
		return
	}
	userC, err := r.Cookie("access_token")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "access token unavailable", err)
		return
	}
	userId, err := auth.ValidateJWT(userC.Value, cfg.secret)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "couldn't validate jwt", err)
		return
	}
	game := cfg.Matches.Matches[c.Value].Online

	err1 := game.Players["white"].Conn.WriteMessage(websocket.TextMessage, []byte("test"))
	err2 := game.Players["black"].Conn.WriteMessage(websocket.TextMessage, []byte("test"))

	if err1 == nil && err2 == nil {
		_, err = fmt.Fprintf(w, `<div id="wait" hx-swap-oob="outerHTML"></div>`)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
			return
		}
	}

	var time int8
	var result string
	var winner string
	if userId == game.Players["white"].ID {
		gamePlayer := game.Players["black"]
		time = gamePlayer.ReconnectTimer
		time -= 1
		gamePlayer.ReconnectTimer = time
		game.Players["black"] = gamePlayer
		result = "1-0"
		winner = "white"
	} else {
		gamePlayer := game.Players["white"]
		time = gamePlayer.ReconnectTimer
		time -= 1
		gamePlayer.ReconnectTimer = time
		game.Players["white"] = gamePlayer
		result = "0-1"
		winner = "black"
	}
	if time < 0 {
		_, err = fmt.Fprintf(w, `<div id="wait" hx-swap-oob="outerHTML"></div>`)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
			return
		}

		err := components.EndGameModal(result, winner).Render(r.Context(), w)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
			return
		}
		return
	}

	_, err = fmt.Fprintf(w, `<span id="waiting" hx-swap-oob="true">%v</span>`, time)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't send time", err)
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
			responses.RespondWithAnError(w, http.StatusInternalServerError, "Couldn't render template", err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) cancelOnlineHandler(w http.ResponseWriter, r *http.Request) {
	currentGame, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}

	userToken, err := r.Cookie("access_token")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "user not found", err)
		return
	}

	userId, err := auth.ValidateJWT(userToken.Value, cfg.secret)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "invalid token", err)
		return
	}

	saveGame, _ := cfg.Matches.GetMatch(currentGame.Value)

	onlineGame := cfg.Matches.Matches[currentGame.Value].Online

	var result string
	if onlineGame.Players["white"].ID == userId {
		result = "0-1"
	} else {
		result = "1-0"
	}

	err = cfg.database.UpdateMatchOnEnd(r.Context(), database.UpdateMatchOnEndParams{
		Result: result,
		ID:     saveGame.MatchId,
	})
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error updating match", err)
		return
	}

	cGC := cfg.removeCookie("current_game")
	http.SetCookie(w, &cGC)

	_, err = fmt.Fprintf(w, `<div id="rec" hx-swap-oob="outerHTML"></div>`)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error sending message", err)
		return
	}
}

func (cfg *appConfig) continueOnlineHandler(w http.ResponseWriter, r *http.Request) {
	currentGame, err := r.Cookie("current_game")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "game not found", err)
		return
	}

	userToken, err := r.Cookie("access_token")
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "user not found", err)
		return
	}

	userId, err := auth.ValidateJWT(userToken.Value, cfg.secret)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "invalid token", err)
		return
	}

	user, err := cfg.database.GetUserById(r.Context(), userId)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "user not found", err)
		return
	}

	match, ok := cfg.Matches.GetMatch(currentGame.Value)

	onlineGame, ok2 := matches.OnlineGame{}, false

	if ok {
		onlineGame, ok2 = match.IsOnlineMatch()
	}

	if !ok || !ok2 {

		cGC := cfg.removeCookie("current_game")
		http.SetCookie(w, &cGC)

		match, _ := cfg.Matches.GetMatch("initial")
		match.FillBoard()

		whitePlayer := components.PlayerStruct{
			Image:  "/assets/images/user-icon.png",
			Name:   user.Name,
			Timer:  utils.FormatTime(match.WhiteTimer),
			Pieces: "white",
		}
		blackPlayer := components.PlayerStruct{
			Image:  "/assets/images/user-icon.png",
			Name:   "Opponent",
			Timer:  utils.FormatTime(match.BlackTimer),
			Pieces: "black",
		}

		err = layout.MainPagePrivate(match.Board, match.Pieces, match.CoordinateMultiplier, whitePlayer, blackPlayer, match.TakenPiecesWhite, match.TakenPiecesBlack, true).Render(r.Context(), w)

		if err != nil {
			responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
			return
		}

		return
	}

	var blackPlayer components.OnlinePlayerStruct
	var whitePlayer components.OnlinePlayerStruct

	for color, player := range onlineGame.Players {
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
		match.Board,
		match.Pieces,
		multiplier,
		whitePlayer,
		blackPlayer,
		match.TakenPiecesWhite,
		match.TakenPiecesBlack,
		true,
	).Render(r.Context(), w)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "websocket upgrade failed", err)
		return
	}
}

func (cfg *appConfig) cancelOnlineSearchHandler(w http.ResponseWriter, r *http.Request) {

	currentGame, err := r.Cookie("current_game")

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "failed getting the cookie", err)
		return
	}

	cGC := cfg.removeCookie("current_game")
	http.SetCookie(w, &cGC)

	delete(cfg.Matches.Matches, currentGame.Value)

	_, err = w.Write([]byte{})
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "failed closing the modal", err)
		return
	}
}
