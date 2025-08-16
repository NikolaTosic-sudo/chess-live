package main

import (
	"fmt"
	"log"
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
		fmt.Println(err)
	}
}

func (cfg *appConfig) closeModalHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{})
}

func (cfg *appConfig) signupModalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := components.Signup().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *appConfig) loginModalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := components.Login().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *appConfig) signupHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	hashedPassword, err := auth.HashedPassword(password)

	if err != nil {
		fmt.Println(err)
		return
	}

	user, err := cfg.database.CreateUser(r.Context(), database.CreateUserParams{
		Name:           name,
		Email:          email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		if strings.Contains(err.Error(), "violates unique constraint") {
			fmt.Fprintf(w, `
				<div id="incorrect-password" hx-swap-oob="innerHTML">
					<p class="text-red-400 text-center">User with that email already exists</p>
				</div>
			`)
		}
		fmt.Println(err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		fmt.Println(err)
		return
	}

	refreshString, err := auth.MakeRefreshToken()

	if err != nil {
		log.Print("couldn't generate refresh token", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		log.Print("couldn't create refresh token", err)
		w.WriteHeader(http.StatusInternalServerError)
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
			fmt.Fprintf(w, `
				<div id="incorrect-password" hx-swap-oob="innerHTML">
					<p class="text-red-400 text-center">User with the email doesn't exist</p>
				</div>
			`)
		}
		fmt.Println(err)
		return
	}

	err = auth.CheckPassword(password, user.HashedPassword)

	if err != nil {
		if strings.Contains(err.Error(), "hashedPassword is not the hash of the given password") {
			fmt.Fprintf(w, `
				<div id="incorrect-password" hx-swap-oob="innerHTML">
					<p class="text-red-400 text-center">Incorrect password</p>
				</div>
			`)
		}
		fmt.Println(err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		fmt.Println(err)
		return
	}

	refreshString, err := auth.MakeRefreshToken()

	if err != nil {
		log.Print("couldn't generate refresh token", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		log.Print("couldn't create refresh token", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		fmt.Println("no token", err)
		w.Header().Add("Hx-Redirect", "/")
		return
	}

	userId, err := auth.ValidateJWT(c.Value, cfg.secret)

	if err != nil {
		fmt.Println("invalid jwt")
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
