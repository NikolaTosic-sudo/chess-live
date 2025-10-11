package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
)

func (cfg *appConfig) loginOpenHandler(w http.ResponseWriter, r *http.Request) {
	err := layout.LoginModal().Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
	}
}

func (cfg *appConfig) closeModalHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte{})
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "failed closing the modal", err)
		return
	}
}

func (cfg *appConfig) signupModalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := components.Signup().Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
	}
}

func (cfg *appConfig) loginModalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := components.Login().Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
	}
}

func (cfg *appConfig) signupHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	hashedPassword, err := auth.HashedPassword(password)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "hashing password failed", err)
		return
	}

	user, err := cfg.database.CreateUser(r.Context(), database.CreateUserParams{
		Name:           name,
		Email:          email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		if strings.Contains(err.Error(), "violates unique constraint") {
			message := "User with that email already exists"
			_, err = fmt.Fprintf(w, getLogErrorMessage(), message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
		respondWithAnError(w, http.StatusInternalServerError, "couldn't create user", err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't make JWT", err)
		return
	}

	refreshString, err := auth.MakeRefreshToken()

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't generate refresh token", err)
		return
	}

	_, err = cfg.database.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshString,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 168),
	})

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't generate refresh token", err)
		return
	}

	c := http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	refreshC := http.Cookie{
		Name:     "refresh_token",
		Value:    refreshString,
		Path:     "/api/refresh",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
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

	sGC := http.Cookie{
		Name:     "saved_game",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &c)
	http.SetCookie(w, &refreshC)
	http.SetCookie(w, &cGC)
	http.SetCookie(w, &sGC)

	cfg.users[user.ID] = User{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}

	w.Header().Add("Hx-Redirect", "/private")
}

func (cfg *appConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := cfg.database.GetUserByEmail(r.Context(), email)

	if err != nil {
		if strings.Contains(err.Error(), "no rows in result") {
			message := "User with the email doesn't exist"
			_, err := fmt.Fprintf(w, getLogErrorMessage(), message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}

		respondWithAnError(w, http.StatusInternalServerError, "couldn't get user by email", err)
		return
	}

	err = auth.CheckPassword(password, user.HashedPassword)

	if err != nil {
		if strings.Contains(err.Error(), "hashedPassword is not the hash of the given password") {
			message := "Incorrect password"
			_, err := fmt.Fprintf(w, getLogErrorMessage(), message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
		respondWithAnError(w, http.StatusInternalServerError, "error with checking the password", err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't make jwt", err)
		return
	}

	refreshString, err := auth.MakeRefreshToken()

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't make refresh token", err)
		return
	}

	_, err = cfg.database.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshString,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 168),
	})

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't make refresh token", err)
		return
	}

	c := http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	refreshC := http.Cookie{
		Name:     "refresh_token",
		Value:    refreshString,
		Path:     "/api/refresh",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
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

	sGC := http.Cookie{
		Name:     "saved_game",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &c)
	http.SetCookie(w, &refreshC)
	http.SetCookie(w, &cGC)
	http.SetCookie(w, &sGC)

	cfg.users[user.ID] = User{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}

	w.Header().Add("Hx-Redirect", "/private")
}

func (cfg *appConfig) logoutHandler(w http.ResponseWriter, r *http.Request) {

	c, err := r.Cookie("access_token")

	if err != nil {
		logError("no token found", err)
		w.Header().Add("Hx-Redirect", "/")
		return
	}

	userId, err := auth.ValidateJWT(c.Value, cfg.secret)

	if err != nil {
		logError("invalid jwt", err)
		w.Header().Add("Hx-Redirect", "/")
		return
	}

	delete(cfg.users, userId)

	accC := http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	refreshC := http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
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
	sGC := http.Cookie{
		Name:     "saved_game",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &accC)
	http.SetCookie(w, &refreshC)
	http.SetCookie(w, &cGC)
	http.SetCookie(w, &sGC)

	w.Header().Add("Hx-Redirect", "/")
}
