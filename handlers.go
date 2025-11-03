package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/NikolaTosic-sudo/chess-live/internal/matches"
	"github.com/NikolaTosic-sudo/chess-live/internal/queue"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/NikolaTosic-sudo/chess-live/internal/utils"
	"github.com/google/uuid"
)

func (cfg *appConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	var game string
	c, err := r.Cookie("current_game")

	if err != nil {
		if !strings.Contains(err.Error(), "named cookie not present") {
			responses.LogError("game not found", err)
		}
		game = "initial"
	} else if c.Value != "" {
		game = c.Value
	} else {
		game = "initial"
	}

	match, ok := cfg.Matches.GetMatch(game)

	if !ok {
		match = cfg.Matches.GetInitialMatch()
	}

	match.FillBoard()

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Guest",
		Timer:  utils.FormatTime(match.WhiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  utils.FormatTime(match.BlackTimer),
		Pieces: "black",
	}

	err = layout.MainPage(match.Board, match.Pieces, match.CoordinateMultiplier, whitePlayer, blackPlayer, match.TakenPiecesWhite, match.TakenPiecesBlack).Render(r.Context(), w)

	if err != nil {
		responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) privateBoardHandler(w http.ResponseWriter, r *http.Request) {
	user, err := cfg.getUser(r)

	if err != nil {
		responses.LogError("error getting user:", err)
	}

	userName := user.Name

	var game string
	c, err := r.Cookie("current_game")

	if err != nil {
		game = "initial"
	} else if c.Value != "" {
		game = c.Value
	} else {
		game = "initial"
	}

	sC, err := r.Cookie("saved_game")

	if err == nil {
		game = sC.Value

		startGame := cfg.makeCookie("current_game", game, "/")

		sGC := cfg.removeCookie("saved_game")

		http.SetCookie(w, &startGame)
		http.SetCookie(w, &sGC)
	}

	match, _ := cfg.Matches.GetMatch(game)
	match.FillBoard()

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   userName,
		Timer:  utils.FormatTime(match.WhiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  utils.FormatTime(match.BlackTimer),
		Pieces: "black",
	}

	err = layout.MainPagePrivate(match.Board, match.Pieces, match.CoordinateMultiplier, whitePlayer, blackPlayer, match.TakenPiecesWhite, match.TakenPiecesBlack, false).Render(r.Context(), w)

	if err != nil {
		responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) onlineBoardHandler(w http.ResponseWriter, r *http.Request) {
	user, err := cfg.getUser(r)
	if err != nil {
		responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Please try again")
		return
	}

	userName := user.Name
	userId := user.ID

	onlineMatches := cfg.Matches.GetAllOnlineMatches()

	if len(onlineMatches) > 0 {
		for gameName, match := range onlineMatches {

			game := match.Online

			if game.PlayersQueue.HasSpot() {

				multiplier, err := cfg.getMultiplier(r)
				if err != nil {
					responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Please try again")
					return
				}

				game.PlayersQueue.Enqueue(components.OnlinePlayerStruct{
					ID:             userId,
					Name:           userName,
					Image:          "/assets/images/user-icon.png",
					Timer:          utils.FormatTime(600),
					ReconnectTimer: 30,
					Multiplier:     multiplier,
				})

				for color := range game.Players {
					playerDq, err := game.PlayersQueue.Dequeue()

					if err != nil {
						responses.RespondWithAnError(w, http.StatusInternalServerError, "", err)
						return
					}

					playerDq.Pieces = color

					player := playerDq

					game.Players[color] = player
				}

				whitePlayer := game.Players["white"]
				blackPlayer := game.Players["black"]

				matchId, _ := cfg.database.CreateMatch(r.Context(), database.CreateMatchParams{
					White:    whitePlayer.Name,
					Black:    blackPlayer.Name,
					FullTime: 600,
					IsOnline: true,
				})

				playersId := []uuid.UUID{whitePlayer.ID, blackPlayer.ID}
				for _, id := range playersId {
					_ = cfg.database.CreateMatchUser(r.Context(), database.CreateMatchUserParams{
						UserID:  id,
						MatchID: matchId,
					})
				}

				startingBoard := MakeBoard()
				startingPieces := MakePieces()

				match.Board = startingBoard
				match.Pieces = startingPieces
				match.SelectedPiece = components.Piece{}
				match.CoordinateMultiplier = multiplier
				match.IsWhiteTurn = true
				match.IsWhiteUnderCheck = false
				match.IsBlackUnderCheck = false
				match.WhiteTimer = 600
				match.BlackTimer = 600
				match.Addition = 0
				match.MatchId = matchId
				match.Online = game

				cfg.Matches.SetMatch(gameName, match)

				startGame := cfg.makeCookie("current_game", gameName, "/")

				match.FillBoard()
				match.UpdateCoordinates(whitePlayer.Multiplier)
				http.SetCookie(w, &startGame)

				err = layout.MainPageOnline(match.Board, match.Pieces, whitePlayer.Multiplier, whitePlayer, blackPlayer, match.TakenPiecesWhite, match.TakenPiecesBlack, false).Render(r.Context(), w)
				if err != nil {
					responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
				}
				return
			}
		}
	}

	var currentGame string

	randomString, err := auth.MakeRefreshToken()
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "coudln't make refresh token", err)
		return
	}
	currentGame = fmt.Sprintf("online:%v", randomString)

	startGame := cfg.makeCookie("current_game", currentGame, "/")

	http.SetCookie(w, &startGame)

	multiplier, err := cfg.getMultiplier(r)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error getting multiplier", err)
		return
	}

	qS := queue.PlayersQueue{}

	pQ := qS.NewQueue()

	pQ.Enqueue(components.OnlinePlayerStruct{
		ID:             userId,
		Name:           userName,
		Image:          "/assets/images/user-icon.png",
		Timer:          utils.FormatTime(600),
		ReconnectTimer: 30,
		Multiplier:     multiplier,
	})

	match := matches.Match{
		IsOnline: true,
		Online: matches.OnlineGame{
			Players: map[string]components.OnlinePlayerStruct{
				"white": {},
				"black": {},
			},
			Message:      make(chan string),
			PlayerMsg:    make(chan string),
			Player:       make(chan components.OnlinePlayerStruct),
			PlayersQueue: pQ,
		},
	}

	cfg.Matches.SetMatch(currentGame, match)

	err = components.WaitingModal().Render(r.Context(), w)
	if err != nil {
		responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "couldn't render template")
		return
	}
}

func (cfg *appConfig) updateMultiplerHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	multiplier, err := strconv.Atoi(r.FormValue("multiplier"))

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return
	}

	c, err := r.Cookie("current_game")
	var currentGame string
	if err != nil {
		currentGame = "initial"
	} else {
		currentGame = c.Value
	}
	match, _ := cfg.Matches.GetMatch(currentGame)

	match.CoordinateMultiplier = multiplier
	match.UpdateCoordinates(multiplier)
	cfg.Matches.SetMatch(currentGame, match)

	multiplierCookie := cfg.makeCookie("multiplier", r.FormValue("multiplier"), "/")

	http.SetCookie(w, &multiplierCookie)

	for k, piece := range match.Pieces {
		tile := match.Board[piece.Tile]

		_, err := fmt.Fprintf(
			w,
			responses.GetSinglePieceMessage(),
			k,
			tile.Coordinates[0],
			tile.Coordinates[1],
			piece.Image,
		)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}
}

func (cfg *appConfig) startGameHandler(w http.ResponseWriter, r *http.Request) {

	user, err := cfg.isUserLoggedIn(r)
	if err != nil {
		if !strings.Contains(err.Error(), "named cookie not present") {
			responses.LogError("user not logged in", err)
		}
	}
	err = r.ParseForm()
	if err != nil {
		responses.LogError("couldn't parse form", err)
	}
	duration := r.FormValue("duration")
	var newGameName string
	var matchId int32
	userName := "Guest"
	if user != uuid.Nil {
		newGameName = user.String()

		fullUser, err := cfg.database.GetUserById(r.Context(), user)
		userName = fullUser.Name

		if err != nil {
			responses.RespondWithAnError(w, http.StatusNotFound, "user not found in db", err)
		} else {
			matchId, err = cfg.database.CreateMatch(r.Context(), database.CreateMatchParams{
				White:    fullUser.Name,
				Black:    "Opponent",
				FullTime: 600,
				IsOnline: false,
				Result:   "0-0",
			})

			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't create match", err)
				return
			}

			err = cfg.database.CreateMatchUser(r.Context(), database.CreateMatchUserParams{
				MatchID: matchId,
				UserID:  fullUser.ID,
			})

			if err != nil {
				responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't create match", err)
				return
			}
		}
	} else {
		randomString, err := auth.MakeRefreshToken()

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't make refresh token", err)
			return
		}

		newGameName = randomString
	}

	startGame := cfg.makeCookie("current_game", newGameName, "/")

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	durationSplit := strings.Split(duration, "+")
	timer, err := strconv.Atoi(durationSplit[0])
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert duration", err)
		return
	}
	addition, err := strconv.Atoi(durationSplit[1])
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert duration", err)
		return
	}

	multiplier, err := cfg.getMultiplier(r)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return
	}

	cur := matches.Match{
		Board:                startingBoard,
		Pieces:               startingPieces,
		SelectedPiece:        components.Piece{},
		CoordinateMultiplier: multiplier,
		IsWhiteTurn:          true,
		IsWhiteUnderCheck:    false,
		IsBlackUnderCheck:    false,
		WhiteTimer:           timer,
		BlackTimer:           timer,
		Addition:             addition,
		MatchId:              matchId,
	}

	cfg.Matches.SetMatch(newGameName, cur)

	cur.FillBoard()
	cur.UpdateCoordinates(cur.CoordinateMultiplier)
	http.SetCookie(w, &startGame)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   userName,
		Timer:  utils.FormatTime(cur.WhiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  utils.FormatTime(cur.BlackTimer),
		Pieces: "black",
	}

	err = components.StartLocalGame(cur.Board, cur.Pieces, multiplier, whitePlayer, blackPlayer).Render(r.Context(), w)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}

}

func (cfg *appConfig) resumeGameHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")

	if err != nil {
		responses.RespondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	match, ok := cfg.Matches.GetMatch(c.Value)

	if !ok {
		responses.RespondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	match.FillBoard()
	match.UpdateCoordinates(match.CoordinateMultiplier)

	err = components.StartGameRight().Render(r.Context(), w)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) getAllMovesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")

	if err != nil {
		responses.RespondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	match, ok := cfg.Matches.GetMatch(c.Value)

	if !ok {
		responses.RespondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	for i := 1; i <= len(match.AllMoves); i++ {
		var message string
		if i%2 == 0 {
			message = fmt.Sprintf(
				responses.GetMovesUpdateMessage(),
				match.AllMoves[i-1],
			)
		} else {
			message = fmt.Sprintf(
				responses.GetMovesNumberUpdateMessage(),
				i/2+1,
				match.AllMoves[i-1],
			)
		}

		err := match.SendMessage(w, message, [2][]int{})
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}

	}
}

func (cfg *appConfig) timeOptionHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprint(w, responses.GetTimePicker())

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}
}

func (cfg *appConfig) setTimeOption(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	time := r.FormValue("time")
	addition := r.FormValue("addition")
	var a int
	if addition != "" {
		a, err = strconv.Atoi(addition)

		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
			return
		}
	}
	t, err := strconv.Atoi(time)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	var seconds string

	if a != 0 {
		seconds = fmt.Sprintf("+ %v sec", a)
	}

	duration := fmt.Sprintf("%v+%v", t*60, a)

	_, err = fmt.Fprintf(
		w,
		responses.GetTimerSwitchMessage(),
		utils.FormatTime(t*60),
		utils.FormatTime(t*60),
		duration,
		time,
		seconds,
	)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}
}

func (cfg *appConfig) matchHistoryHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("access_token")

	if err != nil {
		responses.RespondWithAnError(w, http.StatusNoContent, "game not found", err)
		return
	}

	userId, err := auth.ValidateJWT(c.Value, cfg.secret)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusUnauthorized, "unauthorized user", err)
		return
	}

	dbMatches, err := cfg.database.GetAllMatchesForUser(r.Context(), userId)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "error in database", err)
		return
	}

	var matches []components.MatchStruct

	for i := 0; i < len(dbMatches); i++ {
		numberOfMoves, err := cfg.database.GetNumberOfMovesPerMatch(r.Context(), dbMatches[i].ID)
		if err != nil {
			responses.RespondWithAnError(w, http.StatusInternalServerError, "error in database", err)
			return
		}
		newMatch := components.MatchStruct{
			White:   dbMatches[i].White,
			Black:   dbMatches[i].Black,
			Ended:   dbMatches[i].Ended,
			Date:    dbMatches[i].CreatedAt.Format("Jan 2, 2006"),
			NoMoves: int(numberOfMoves),
			Result:  dbMatches[i].Result,
			Online:  dbMatches[i].IsOnline,
			MatchId: int(dbMatches[i].ID),
		}

		matches = append(matches, newMatch)
	}

	err = components.MatchHistory(matches).Render(r.Context(), w)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) playHandler(w http.ResponseWriter, r *http.Request) {
	var userName string
	userC, err := r.Cookie("access_token")

	if err != nil {
		responses.LogError("user not found", err)
		userName = "Guest"
	} else if userC.Value != "" {
		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			responses.LogError("user not found", err)
			userName = "Guest"
		}

		user, err := cfg.database.GetUserById(r.Context(), userId)

		if err != nil {
			responses.LogError("user not found", err)
			userName = "Guest"
		} else if user.Name != "" {
			userName = user.Name
		}
	} else {
		userName = "Guest"
	}

	var game string
	c, err := r.Cookie("current_game")

	if err != nil {
		game = "initial"
	} else if strings.Contains(c.Value, "database:") {
		cGC := cfg.removeCookie("current_game")

		http.SetCookie(w, &cGC)
		game = "initial"
	} else if c.Value != "" {
		game = c.Value
	} else {
		game = "initial"
	}

	sC, err := r.Cookie("saved_game")

	if err == nil {
		game = sC.Value

		startGame := cfg.makeCookie("current_game", game, "/")

		sGC := cfg.removeCookie("saved_game")

		http.SetCookie(w, &startGame)
		http.SetCookie(w, &sGC)
	}

	match, _ := cfg.Matches.GetMatch(game)
	match.FillBoard()

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   userName,
		Timer:  utils.FormatTime(match.WhiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  utils.FormatTime(match.BlackTimer),
		Pieces: "black",
	}

	err = layout.MainPagePrivate(match.Board, match.Pieces, match.CoordinateMultiplier, whitePlayer, blackPlayer, match.TakenPiecesWhite, match.TakenPiecesBlack, false).Render(r.Context(), w)

	if err != nil {
		responses.RespondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) matchesHandler(w http.ResponseWriter, r *http.Request) {
	strId := r.PathValue("id")
	id, err := strconv.Atoi(strId)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
		return
	}

	match, err := cfg.database.GetMatchById(r.Context(), int32(id))

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't get match", err)
		return
	}

	newGame := fmt.Sprintf("database:matchId-%v", match.ID)

	c, noCookie := r.Cookie("current_game")

	if noCookie == nil && c.Value != "" {
		saveGame := cfg.makeCookie("saved_game", c.Value, "/")

		http.SetCookie(w, &saveGame)
	}

	multiplier, err := cfg.getMultiplier(r)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
		return
	}

	startGame := cfg.makeCookie("current_game", newGame, "/")

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	cur := matches.Match{
		Board:                startingBoard,
		Pieces:               startingPieces,
		SelectedPiece:        components.Piece{},
		CoordinateMultiplier: multiplier,
		IsWhiteTurn:          true,
		IsWhiteUnderCheck:    false,
		IsBlackUnderCheck:    false,
		WhiteTimer:           int(match.FullTime),
		BlackTimer:           int(match.FullTime),
		MatchId:              match.ID,
	}

	cfg.Matches.SetMatch(newGame, cur)

	cur.FillBoard()
	cur.UpdateCoordinates(cur.CoordinateMultiplier)
	http.SetCookie(w, &startGame)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   match.White,
		Timer:  utils.FormatTime(int(match.FullTime)),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   match.Black,
		Timer:  utils.FormatTime(int(match.FullTime)),
		Pieces: "black",
	}

	moves, err := cfg.database.GetAllMovesForMatch(r.Context(), match.ID)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't get all moves", err)
		return
	}

	err = layout.MatchHistoryBoard(cur.Board, cur.Pieces, cur.CoordinateMultiplier, whitePlayer, blackPlayer, cur.TakenPiecesWhite, cur.TakenPiecesBlack, moves).Render(r.Context(), w)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) moveHistoryHandler(w http.ResponseWriter, r *http.Request) {
	tile := r.PathValue("tile")
	c, err := r.Cookie("current_game")

	if err != nil {
		responses.RespondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}

	cookie := strings.Split(c.Value, "-")

	matchId, err := strconv.Atoi(cookie[1])

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
		return
	}

	board, err := cfg.database.GetBoardForMove(r.Context(), database.GetBoardForMoveParams{
		MatchID: int32(matchId),
		Move:    tile,
	})

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't get the board for the move", err)
		return
	}

	var boardState map[string]string

	err = json.Unmarshal(board.Board, &boardState)

	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't unmarshal board state", err)
		return
	}

	startingPieces := MakePieces()

	pieces := make(map[string]components.Piece, 0)

	for k, v := range boardState {
		curr := startingPieces[k]
		curr.Tile = v
		pieces[k] = curr
	}
	curr, _ := cfg.Matches.GetMatch(c.Value)

	curr.CleanFillBoard(pieces)

	err = components.UpdateBoardHistory(curr.Board, pieces, curr.CoordinateMultiplier, utils.FormatTime(int(board.WhiteTime)), utils.FormatTime(int(board.BlackTime))).Render(r.Context(), w)
	if err != nil {
		responses.RespondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) endModalHandler(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusNoContent)
}
