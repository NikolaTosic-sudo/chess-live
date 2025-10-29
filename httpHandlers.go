package main

import (
	"net/http"
	"strings"
)

type handler struct {
	method    string
	reqPath   string
	handlFunc func(http.ResponseWriter, *http.Request)
}

func (cfg *appConfig) registerAllHandlers() {
	var handlers = []handler{
		{
			method:    "",
			reqPath:   "/",
			handlFunc: cfg.middleWareCheckForUser(cfg.boardHandler),
		},
		{
			method:    "GET",
			reqPath:   "/private",
			handlFunc: cfg.middleWareCheckForUserPrivate(cfg.privateBoardHandler),
		},
		{
			method:    "POST",
			reqPath:   "/start",
			handlFunc: cfg.startGameHandler,
		},
		{
			method:    "POST",
			reqPath:   "/resume",
			handlFunc: cfg.resumeGameHandler,
		},
		{
			method:    "POST",
			reqPath:   "/move",
			handlFunc: cfg.moveHandler,
		},
		{
			method:    "POST",
			reqPath:   "/move-to",
			handlFunc: cfg.moveToHandler,
		},
		{
			method:    "POST",
			reqPath:   "/cover-check",
			handlFunc: cfg.coverCheckHandler,
		},
		{
			method:    "GET",
			reqPath:   "/timer",
			handlFunc: cfg.timerHandler,
		},
		{
			method:    "GET",
			reqPath:   "/time-options",
			handlFunc: cfg.timeOptionHandler,
		},
		{
			method:    "POST",
			reqPath:   "/set-time",
			handlFunc: cfg.setTimeOption,
		},
		{
			method:    "POST",
			reqPath:   "/update-multiplier",
			handlFunc: cfg.updateMultiplerHandler,
		},
		{
			method:    "GET",
			reqPath:   "/login",
			handlFunc: cfg.loginOpenHandler,
		},
		{
			method:    "GET",
			reqPath:   "/logout",
			handlFunc: cfg.logoutHandler,
		},
		{
			method:    "GET",
			reqPath:   "/close-modal",
			handlFunc: cfg.closeModalHandler,
		},
		{
			method:    "GET",
			reqPath:   "/login-modal",
			handlFunc: cfg.loginModalHandler,
		},
		{
			method:    "GET",
			reqPath:   "/signup-modal",
			handlFunc: cfg.signupModalHandler,
		},
		{
			method:    "POST",
			reqPath:   "/auth-signup",
			handlFunc: cfg.signupHandler,
		},
		{
			method:    "POST",
			reqPath:   "/auth-login",
			handlFunc: cfg.loginHandler,
		},
		{
			method:    "GET",
			reqPath:   "/api/refresh",
			handlFunc: cfg.refreshToken,
		},
		{
			method:    "GET",
			reqPath:   "/all-moves",
			handlFunc: cfg.getAllMovesHandler,
		},
		{
			method:    "GET",
			reqPath:   "/match-history",
			handlFunc: cfg.middleWareCheckForUserPrivate(cfg.matchHistoryHandler),
		},
		{
			method:    "GET",
			reqPath:   "/play-game",
			handlFunc: cfg.playHandler,
		},
		{
			method:    "GET",
			reqPath:   "/matches/{id}",
			handlFunc: cfg.matchesHandler,
		},
		{
			method:    "GET",
			reqPath:   "/move-history/{tile}",
			handlFunc: cfg.moveHistoryHandler,
		},
		{
			method:    "POST",
			reqPath:   "/promotion",
			handlFunc: cfg.handlePromotion,
		},
		{
			method:    "GET",
			reqPath:   "/online",
			handlFunc: cfg.wsHandler,
		},
		{
			method:    "GET",
			reqPath:   "/play-online",
			handlFunc: cfg.middleWareCheckForUserPrivate(cfg.onlineBoardHandler),
		},
		{
			method:    "GET",
			reqPath:   "/searching",
			handlFunc: cfg.searchingOppHandler,
		},
		{
			method:    "GET",
			reqPath:   "/end-game",
			handlFunc: cfg.endGameHandler,
		},
		{
			method:    "GET",
			reqPath:   "/surrender",
			handlFunc: cfg.surrenderHandler,
		},
		{
			method:    "GET",
			reqPath:   "/wait-reconnect",
			handlFunc: cfg.waitingForReconnect,
		},
		{
			method:    "GET",
			reqPath:   "/check-online",
			handlFunc: cfg.checkOnlineHandler,
		},
		{
			method:    "GET",
			reqPath:   "/cancel-online",
			handlFunc: cfg.cancelOnlineHandler,
		},
		{
			method:    "GET",
			reqPath:   "/continue-online",
			handlFunc: cfg.continueOnlineHandler,
		},
		{
			method:    "GET",
			reqPath:   "/handle-end",
			handlFunc: cfg.endModalHandler,
		},
		{
			method:    "GET",
			reqPath:   "/cancel-online-search",
			handlFunc: cfg.cancelOnlineSearchHandler,
		},
	}

	for _, h := range handlers {
		reqLine := strings.Join([]string{h.method, h.reqPath}, " ")

		http.HandleFunc(reqLine, h.handlFunc)
	}
}
