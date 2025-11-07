package main

import (
	"net/http"
	"strings"
)

type handler struct {
	method     string
	reqPath    string
	handleFunc func(http.ResponseWriter, *http.Request)
}

func (cfg *appConfig) registerAllHandlers() {
	var handlers = []handler{
		{
			method:     "",
			reqPath:    "/",
			handleFunc: cfg.middleWareCheckForUser(cfg.boardHandler),
		},
		{
			method:     "GET",
			reqPath:    "/private",
			handleFunc: cfg.middleWareCheckForUserPrivate(cfg.privateBoardHandler),
		},
		{
			method:     "POST",
			reqPath:    "/start",
			handleFunc: cfg.startGameHandler,
		},
		{
			method:     "POST",
			reqPath:    "/resume",
			handleFunc: cfg.resumeGameHandler,
		},
		{
			method:     "POST",
			reqPath:    "/move",
			handleFunc: cfg.moveHandler,
		},
		{
			method:     "POST",
			reqPath:    "/move-to",
			handleFunc: cfg.moveToHandler,
		},
		{
			method:     "POST",
			reqPath:    "/cover-check",
			handleFunc: cfg.coverCheckHandler,
		},
		{
			method:     "GET",
			reqPath:    "/timer",
			handleFunc: cfg.timerHandler,
		},
		{
			method:     "GET",
			reqPath:    "/time-options",
			handleFunc: cfg.timeOptionHandler,
		},
		{
			method:     "POST",
			reqPath:    "/set-time",
			handleFunc: cfg.setTimeOption,
		},
		{
			method:     "POST",
			reqPath:    "/update-multiplier",
			handleFunc: cfg.updateMultiplerHandler,
		},
		{
			method:     "GET",
			reqPath:    "/login",
			handleFunc: cfg.loginOpenHandler,
		},
		{
			method:     "GET",
			reqPath:    "/logout",
			handleFunc: cfg.logoutHandler,
		},
		{
			method:     "GET",
			reqPath:    "/close-modal",
			handleFunc: cfg.closeModalHandler,
		},
		{
			method:     "GET",
			reqPath:    "/login-modal",
			handleFunc: cfg.loginModalHandler,
		},
		{
			method:     "GET",
			reqPath:    "/signup-modal",
			handleFunc: cfg.signupModalHandler,
		},
		{
			method:     "POST",
			reqPath:    "/auth-signup",
			handleFunc: cfg.signupHandler,
		},
		{
			method:     "POST",
			reqPath:    "/auth-login",
			handleFunc: cfg.loginHandler,
		},
		{
			method:     "GET",
			reqPath:    "/api/refresh",
			handleFunc: cfg.refreshToken,
		},
		{
			method:     "GET",
			reqPath:    "/all-moves",
			handleFunc: cfg.getAllMovesHandler,
		},
		{
			method:     "GET",
			reqPath:    "/match-history",
			handleFunc: cfg.middleWareCheckForUserPrivate(cfg.matchHistoryHandler),
		},
		{
			method:     "GET",
			reqPath:    "/play-game",
			handleFunc: cfg.playHandler,
		},
		{
			method:     "GET",
			reqPath:    "/matches/{id}",
			handleFunc: cfg.matchesHandler,
		},
		{
			method:     "GET",
			reqPath:    "/move-history/{tile}",
			handleFunc: cfg.moveHistoryHandler,
		},
		{
			method:     "POST",
			reqPath:    "/promotion",
			handleFunc: cfg.handlePromotion,
		},
		{
			method:     "GET",
			reqPath:    "/online",
			handleFunc: cfg.wsHandler,
		},
		{
			method:     "GET",
			reqPath:    "/play-online",
			handleFunc: cfg.middleWareCheckForUserPrivate(cfg.onlineBoardHandler),
		},
		{
			method:     "GET",
			reqPath:    "/searching",
			handleFunc: cfg.searchingOppHandler,
		},
		{
			method:     "GET",
			reqPath:    "/end-game",
			handleFunc: cfg.endGameHandler,
		},
		{
			method:     "GET",
			reqPath:    "/surrender",
			handleFunc: cfg.surrenderHandler,
		},
		{
			method:     "GET",
			reqPath:    "/offer-draw",
			handleFunc: cfg.offerDrawHandler,
		},
		{
			method:     "GET",
			reqPath:    "/decline-draw",
			handleFunc: cfg.declineDrawHandler,
		},
		{
			method:     "GET",
			reqPath:    "/accept-draw",
			handleFunc: cfg.accpetDrawHandler,
		},
		{
			method:     "GET",
			reqPath:    "/wait-reconnect",
			handleFunc: cfg.waitingForReconnect,
		},
		{
			method:     "GET",
			reqPath:    "/check-online",
			handleFunc: cfg.checkOnlineHandler,
		},
		{
			method:     "GET",
			reqPath:    "/cancel-online",
			handleFunc: cfg.cancelOnlineHandler,
		},
		{
			method:     "GET",
			reqPath:    "/continue-online",
			handleFunc: cfg.continueOnlineHandler,
		},
		{
			method:     "GET",
			reqPath:    "/handle-end",
			handleFunc: cfg.endModalHandler,
		},
		{
			method:     "GET",
			reqPath:    "/cancel-online-search",
			handleFunc: cfg.cancelOnlineSearchHandler,
		},
	}

	for _, h := range handlers {
		reqLine := strings.Join([]string{h.method, h.reqPath}, " ")

		http.HandleFunc(reqLine, h.handleFunc)
	}
}
