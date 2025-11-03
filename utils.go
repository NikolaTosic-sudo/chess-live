package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/NikolaTosic-sudo/chess-live/internal/matches"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/google/uuid"
)

func (cfg *appConfig) refreshToken(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("refresh_token")

	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "refresh token not found", err)
		return
	}

	dbToken, err := cfg.database.SearchForToken(r.Context(), c.Value)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "refresh token not found", err)
		return
	}

	if dbToken.ExpiresAt.Before(time.Now()) {
		delete(cfg.users, dbToken.UserID)
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	user, err := cfg.database.GetUserById(r.Context(), dbToken.UserID)

	if err != nil {
		responses.LogError("no user with that id", err)
		http.Redirect(w, r, "/", http.StatusUnauthorized)
		return
	}

	newToken, err := auth.MakeJWT(user.ID, cfg.secret)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't make jwt", err)
		return
	}

	newC := cfg.makeCookieMaxAge("access_token", newToken, "/", 3600)

	http.SetCookie(w, &newC)

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) checkUser(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie("access_token")

	if err != nil {
		return err
	} else if c.Value != "" {
		userId, err := auth.ValidateJWT(c.Value, cfg.secret)

		if err != nil {
			if strings.Contains(err.Error(), "token is expired") {
				return err
			}
			return err
		} else if userId != uuid.Nil {
			_, err := cfg.database.GetUserById(r.Context(), userId)

			if err != nil {
				return err
			}

			_, ok := cfg.users[userId]

			if ok {
				http.Redirect(w, r, "/private", http.StatusSeeOther)
			}
		}
	}
	return nil
}

func (cfg *appConfig) checkUserPrivate(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie("access_token")
	if err != nil {
		return err
	} else if c.Value != "" {
		userId, err := auth.ValidateJWT(c.Value, cfg.secret)

		if err != nil {
			responses.LogError("user not found", err)
			http.Redirect(w, r, "/", http.StatusFound)
		} else if userId != uuid.Nil {
			_, err := cfg.database.GetUserById(r.Context(), userId)
			if err != nil {
				return err
			}
			_, ok := cfg.users[userId]
			if !ok {
				http.Redirect(w, r, "/", http.StatusFound)
			}
		}
	}
	return nil
}

func (cfg *appConfig) middleWareCheckForUser(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := cfg.checkUser(w, r)
		if err != nil {
			if !strings.Contains(err.Error(), "named cookie not present") {
				responses.LogError("error with check user", err)
			}
		}
		next(w, r)
	})
}

func (cfg *appConfig) middleWareCheckForUserPrivate(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := cfg.checkUserPrivate(w, r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		next(w, r)
	})
}

func (cfg *appConfig) isUserLoggedIn(r *http.Request) (uuid.UUID, error) {
	userId, err := cfg.getUserId(r)

	if err != nil {
		return uuid.Nil, err
	}

	_, err = cfg.database.GetUserById(r.Context(), userId)
	if err != nil {
		return uuid.Nil, err
	}
	_, ok := cfg.users[userId]
	if !ok {
		return uuid.Nil, err
	}

	return userId, nil
}

func (cfg *appConfig) showMoves(match matches.Match, squareName, pieceName string, w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie("current_game")
	if err != nil {
		return err
	}
	if c.Value != "" {
		if strings.Split(c.Value, ":")[0] == "database" {
			return err
		}
	}

	boardState := make(map[string]string, 0)
	for k, v := range match.Pieces {
		boardState[k] = v.Tile
	}

	jsonBoard, err := json.Marshal(boardState)

	if err != nil {
		return err
	}

	userId, err := cfg.isUserLoggedIn(r)
	if err != nil && !strings.Contains(err.Error(), "named cookie not present") {
		return err
	}

	if userId != uuid.Nil {
		err = cfg.database.CreateMove(r.Context(), database.CreateMoveParams{
			Board:     jsonBoard,
			Move:      fmt.Sprintf("%v:%v", pieceName, squareName),
			WhiteTime: int32(match.WhiteTimer),
			BlackTime: int32(match.BlackTimer),
			MatchID:   match.MatchId,
		})

		if err != nil {
			return err
		}
	}

	var message string
	if len(match.AllMoves)%2 == 0 {
		message = fmt.Sprintf(
			responses.GetMovesUpdateMessage(),
			squareName,
		)
	} else {
		message = fmt.Sprintf(
			responses.GetMovesNumberUpdateMessage(),
			len(match.AllMoves)/2+1,
			squareName,
		)
	}

	cfg.Matches.SetMatch(c.Value, match)

	err = match.SendMessage(w, message, [2][]int{})

	return err
}
