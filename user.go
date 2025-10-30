package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
)

func (cfg *appConfig) getUser(r *http.Request) (database.User, error) {
	user := database.User{}

	userC, err := r.Cookie("access_token")

	if err != nil || userC.Value == "" {
		return user, fmt.Errorf("invalid access token")
	}

	userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

	if err != nil {
		return user, err
	}

	user, err = cfg.database.GetUserById(r.Context(), userId)

	if err != nil {
		logError("user not found in the database", err)
		return user, err
	}

	return user, nil
}

func (cfg *appConfig) removeCookiePath(name, path string) http.Cookie {
	cookie := http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie
}

func (cfg *appConfig) removeCookie(name string) http.Cookie {
	cookie := cfg.removeCookiePath(name, "/")

	return cookie
}

func (cfg *appConfig) makeCookieMaxAge(name, value, path string, maxAge int) http.Cookie {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie
}

func (cfg *appConfig) makeCookie(name, value, path string) http.Cookie {
	cookie := cfg.makeCookieMaxAge(name, value, path, 604800)

	return cookie
}

func (cfg *appConfig) getMultiplier(r *http.Request) (int, error) {
	mC, noMc := r.Cookie("multiplier")

	multiplier := 0
	if noMc != nil {
		multiplier = 80
	} else {
		mcInt, err := strconv.Atoi(mC.Value)
		if err != nil {
			return multiplier, err
		}
		multiplier = mcInt
	}

	return multiplier, nil
}
