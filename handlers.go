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
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (cfg *appConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	var game string
	c, err := r.Cookie("current_game")

	if err != nil {
		if !strings.Contains(err.Error(), "named cookie not present") {
			logError("game not found", err)
		}
		game = "initial"
	} else if c.Value != "" {
		game = c.Value
	} else {
		game = "initial"
	}

	match, ok := cfg.Matches[game]
	if !ok {
		game = "initial"
		match = cfg.Matches["initial"]
	}
	cfg.fillBoard(game)
	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Guest",
		Timer:  formatTime(match.whiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  formatTime(match.blackTimer),
		Pieces: "black",
	}

	err = layout.MainPage(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack).Render(r.Context(), w)

	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) privateBoardHandler(w http.ResponseWriter, r *http.Request) {
	var userName string
	userC, err := r.Cookie("access_token")

	if err != nil {
		logError("user not found", err)
		userName = "Guest"
	} else if userC.Value != "" {
		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			userName = "Guest"
		}

		user, err := cfg.database.GetUserById(r.Context(), userId)

		if err != nil {
			logError("user not found in the database", err)
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
	} else if c.Value != "" && !strings.Contains(c.Value, "online:") {
		game = c.Value
	} else if strings.Contains(c.Value, "online:") {
		err := cfg.endGameCleaner(w, r, c.Value)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "end game cleaner failed", err)
			return
		}
		game = "initial"
	} else {
		game = "initial"
	}

	sC, err := r.Cookie("saved_game")

	if err == nil {
		game = sC.Value

		startGame := http.Cookie{
			Name:     "current_game",
			Value:    game,
			Path:     "/",
			MaxAge:   604800,
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

		http.SetCookie(w, &startGame)
		http.SetCookie(w, &sGC)
	}

	match := cfg.Matches[game]
	cfg.fillBoard(game)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   userName,
		Timer:  formatTime(match.whiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  formatTime(match.blackTimer),
		Pieces: "black",
	}

	err = layout.MainPagePrivate(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack).Render(r.Context(), w)

	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) onlineBoardHandler(w http.ResponseWriter, r *http.Request) {
	var userName string
	var userId uuid.UUID
	userCookie, err := r.Cookie("access_token")
	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusNotFound, "No user found")
		return
	} else if userCookie.Value != "" {
		userId, err = auth.ValidateJWT(userCookie.Value, cfg.secret)

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusNotFound, "No user found")
			return
		}

		user, err := cfg.database.GetUserById(r.Context(), userId)

		if err != nil {
			respondWithAnErrorPage(w, r, http.StatusNotFound, "No user found")
			return
		} else if user.Name != "" {
			userName = user.Name
		} else {
			respondWithAnErrorPage(w, r, http.StatusNotFound, "No user found")
			return
		}
	}

	var emptyPlayer components.OnlinePlayerStruct
	if len(cfg.connections) > 0 {
		for gameName, players := range cfg.connections {
			for color, player := range players {
				if player == emptyPlayer {
					connection := cfg.connections[gameName]
					player = components.OnlinePlayerStruct{
						ID:     userId,
						Name:   userName,
						Image:  "/assets/images/user-icon.png",
						Timer:  formatTime(600),
						Pieces: "black",
					}
					connection[color] = player
					cfg.connections[gameName] = connection
					whitePlayer := connection["white"]

					matchId, _ := cfg.database.CreateMatch(r.Context(), database.CreateMatchParams{
						White:    whitePlayer.Name,
						Black:    player.Name,
						FullTime: 600,
						UserID:   userId,
						IsOnline: true,
					})

					startingBoard := MakeBoard()
					startingPieces := MakePieces()

					mC, noMc := r.Cookie("multiplier")

					var multiplier int
					if noMc != nil {
						multiplier = 80
					} else {
						mcInt, err := strconv.Atoi(mC.Value)
						if err != nil {
							respondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
							return
						}
						multiplier = mcInt
					}

					cfg.Matches[gameName] = Match{
						board:                startingBoard,
						pieces:               startingPieces,
						selectedPiece:        components.Piece{},
						coordinateMultiplier: multiplier,
						isWhiteTurn:          true,
						isWhiteUnderCheck:    false,
						isBlackUnderCheck:    false,
						whiteTimer:           600,
						blackTimer:           600,
						addition:             0,
						matchId:              matchId,
					}

					startGame := http.Cookie{
						Name:     "current_game",
						Value:    gameName,
						Path:     "/",
						MaxAge:   604800,
						HttpOnly: true,
						Secure:   true,
						SameSite: http.SameSiteLaxMode,
					}

					match := cfg.Matches[gameName]
					cfg.fillBoard(gameName)
					UpdateCoordinates(&match)
					http.SetCookie(w, &startGame)

					err = layout.MainPageOnline(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, player, match.takenPiecesWhite, match.takenPiecesBlack, false).Render(r.Context(), w)
					if err != nil {
						respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
					}
					return
				}
			}
		}
	}
	var currentGame string

	randomString, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "coudln't make refresh token", err)
		return
	}
	currentGame = fmt.Sprintf("online:%v", randomString)

	startGame := http.Cookie{
		Name:     "current_game",
		Value:    currentGame,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &startGame)
	cfg.connections[currentGame] = map[string]components.OnlinePlayerStruct{
		"white": {
			ID:     userId,
			Name:   userName,
			Image:  "/assets/images/user-icon.png",
			Timer:  formatTime(600),
			Pieces: "white",
		},
		"black": {},
	}

	err = components.WaitingModal().Render(r.Context(), w)
	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "couldn't render template")
		return
	}
}

func (cfg *appConfig) updateMultiplerHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	multiplier, err := strconv.Atoi(r.FormValue("multiplier"))

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return
	}

	c, err := r.Cookie("current_game")
	var currentGame string
	if err != nil {
		currentGame = "initial"
	} else {
		currentGame = c.Value
	}
	match := cfg.Matches[currentGame]

	match.coordinateMultiplier = multiplier
	UpdateCoordinates(&match)
	cfg.Matches[currentGame] = match

	multiplierCookie := http.Cookie{
		Name:     "multiplier",
		Value:    r.FormValue("multiplier"),
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &multiplierCookie)

	for k, piece := range match.pieces {
		tile := match.board[piece.Tile]

		_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile-md tile hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			k, tile.Coordinates[0], tile.Coordinates[1], piece.Image)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
			return
		}
	}
}

func (cfg *appConfig) startGameHandler(w http.ResponseWriter, r *http.Request) {

	user, err := cfg.isUserLoggedIn(r)
	if err != nil {
		if !strings.Contains(err.Error(), "named cookie not present") {
			logError("user not logged in", err)
		}
	}
	err = r.ParseForm()
	if err != nil {
		logError("couldn't parse form", err)
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
			respondWithAnError(w, http.StatusNotFound, "user not found in db", err)
		} else {
			matchId, err = cfg.database.CreateMatch(r.Context(), database.CreateMatchParams{
				White:    fullUser.Name,
				Black:    "Opponent",
				FullTime: 600,
				UserID:   user,
				IsOnline: false,
				Result:   "0-0",
			})

			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't create match", err)
				return
			}
		}
	} else {
		randomString, err := auth.MakeRefreshToken()

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't make refresh token", err)
			return
		}

		newGameName = randomString
	}

	startGame := http.Cookie{
		Name:     "current_game",
		Value:    newGameName,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	durationSplit := strings.Split(duration, "+")
	timer, err := strconv.Atoi(durationSplit[0])
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert duration", err)
		return
	}
	addition, err := strconv.Atoi(durationSplit[1])
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert duration", err)
		return
	}
	mC, noMc := r.Cookie("multiplier")

	var multiplier int
	if noMc != nil {
		multiplier = 80
	} else {
		mcInt, err := strconv.Atoi(mC.Value)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
			return
		}
		multiplier = mcInt
	}

	cfg.Matches[newGameName] = Match{
		board:                startingBoard,
		pieces:               startingPieces,
		selectedPiece:        components.Piece{},
		coordinateMultiplier: multiplier,
		isWhiteTurn:          true,
		isWhiteUnderCheck:    false,
		isBlackUnderCheck:    false,
		whiteTimer:           timer,
		blackTimer:           timer,
		addition:             addition,
		matchId:              matchId,
	}

	cur := cfg.Matches[newGameName]

	cfg.fillBoard(newGameName)
	UpdateCoordinates(&cur)
	http.SetCookie(w, &startGame)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   userName,
		Timer:  formatTime(cur.whiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  formatTime(cur.blackTimer),
		Pieces: "black",
	}

	err = components.StartLocalGame(cur.board, cur.pieces, multiplier, whitePlayer, blackPlayer).Render(r.Context(), w)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}

}

func (cfg *appConfig) resumeGameHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")

	if err != nil {
		respondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	match, ok := cfg.Matches[c.Value]

	if !ok {
		respondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	cfg.fillBoard(c.Value)
	UpdateCoordinates(&match)

	err = components.StartGameRight().Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) getAllMovesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")

	if err != nil {
		respondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	match, ok := cfg.Matches[c.Value]
	onlineGame, found := cfg.connections[c.Value]

	if !ok {
		respondWithAnError(w, http.StatusNoContent, "no game found", err)
		return
	}

	for i := 1; i <= len(match.allMoves); i++ {
		var message string
		if i%2 == 0 {
			message = fmt.Sprintf(`
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v</span>
				</div>
			`,
				match.allMoves[i-1],
			)
		} else {
			message = fmt.Sprintf(`
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v.</span>
					<span>%v</span>
				</div>
		`,
				i/2+1,
				match.allMoves[i-1],
			)
		}
		if found {
			for playerColor, onlinePlayer := range onlineGame {
				err := onlinePlayer.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					respondWithAnError(w, http.StatusInternalServerError, fmt.Sprintf("WebSocket write error to: %v", playerColor), err)
				}
			}
		} else {
			_, err := fmt.Fprint(w, message)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
				return
			}
		}
	}
}

func (cfg *appConfig) timeOptionHandler(w http.ResponseWriter, r *http.Request) {

	_, err := fmt.Fprintf(w, `
		<div class="absolute right-0 mt-2 w-48 bg-[#1e1c1a] border border-[#3a3733] text-white rounded-md shadow-lg z-50">
			<div hx-post="/set-time" hx-vals='{"time": "15"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">15 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "15", "addition": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">15 + 3</div>
			<div hx-post="/set-time" hx-vals='{"time": "10"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">10 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "10", "addition": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">10 + 3</div>
			<div hx-post="/set-time" hx-vals='{"time": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">3 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "3", "addition": "1"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">3 + 1</div>
		</div>
	`)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}
}

func (cfg *appConfig) setTimeOption(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	time := r.FormValue("time")
	addition := r.FormValue("addition")
	var a int
	if addition != "" {
		a, err = strconv.Atoi(addition)

		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
			return
		}
	}
	t, err := strconv.Atoi(time)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return
	}

	var seconds string

	if a != 0 {
		seconds = fmt.Sprintf("+ %v sec", a)
	}

	duration := fmt.Sprintf("%v+%v", t*60, a)

	_, err = fmt.Fprintf(w, `
		<div id="dropdown-menu" hx-swap-oob="true" class="relative mb-8"></div>

		<div id="white" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		<div id="black" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		<input type="hidden" id="timer-value" name="duration" hx-swap-oob="true" value="%v" />

		%v Min %v
	`, formatTime(t*60), formatTime(t*60), duration, time, seconds)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't write to page", err)
		return
	}
}

func (cfg *appConfig) matchHistoryHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("access_token")

	if err != nil {
		respondWithAnError(w, http.StatusNoContent, "game not found", err)
		return
	}

	userId, err := auth.ValidateJWT(c.Value, cfg.secret)

	if err != nil {
		respondWithAnError(w, http.StatusUnauthorized, "unauthorized user", err)
		return
	}

	dbMatches, err := cfg.database.GetAllMatchesForUser(r.Context(), userId)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "error in database", err)
		return
	}

	var matches []components.MatchStruct

	for i := 0; i < len(dbMatches); i++ {
		numberOfMoves, err := cfg.database.GetNumberOfMovesPerMatch(r.Context(), dbMatches[i].ID)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "error in database", err)
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
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) playHandler(w http.ResponseWriter, r *http.Request) {
	var userName string
	userC, err := r.Cookie("access_token")

	if err != nil {
		logError("user not found", err)
		userName = "Guest"
	} else if userC.Value != "" {
		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			logError("user not found", err)
			userName = "Guest"
		}

		user, err := cfg.database.GetUserById(r.Context(), userId)

		if err != nil {
			logError("user not found", err)
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
		logError("no game found", err)
		game = "initial"
	} else if strings.Contains(c.Value, "database:") {
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
		game = "initial"
	} else if c.Value != "" {
		game = c.Value
	} else {
		game = "initial"
	}

	sC, err := r.Cookie("saved_game")

	if err == nil {
		game = sC.Value

		startGame := http.Cookie{
			Name:     "current_game",
			Value:    game,
			Path:     "/",
			MaxAge:   604800,
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

		http.SetCookie(w, &startGame)
		http.SetCookie(w, &sGC)
	}

	match := cfg.Matches[game]
	cfg.fillBoard(game)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   userName,
		Timer:  formatTime(match.whiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  formatTime(match.blackTimer),
		Pieces: "black",
	}

	err = layout.MainPagePrivate(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer, match.takenPiecesWhite, match.takenPiecesBlack).Render(r.Context(), w)

	if err != nil {
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) matchesHandler(w http.ResponseWriter, r *http.Request) {
	strId := r.PathValue("id")
	id, err := strconv.Atoi(strId)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
		return
	}

	match, err := cfg.database.GetMatchById(r.Context(), int32(id))

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't get match", err)
		return
	}

	newGame := fmt.Sprintf("database:matchId-%v", match.ID)

	c, noCookie := r.Cookie("current_game")

	if noCookie == nil {
		saveGame := http.Cookie{
			Name:     "saved_game",
			Value:    c.Value,
			Path:     "/",
			MaxAge:   604800,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, &saveGame)
	}

	mC, noMc := r.Cookie("multiplier")

	var multiplier int
	if noMc != nil {
		multiplier = 80
	} else {
		mcInt, err := strconv.Atoi(mC.Value)
		if err != nil {
			respondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
			return
		}
		multiplier = mcInt
	}

	startGame := http.Cookie{
		Name:     "current_game",
		Value:    newGame,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	cfg.Matches[newGame] = Match{
		board:                startingBoard,
		pieces:               startingPieces,
		selectedPiece:        components.Piece{},
		coordinateMultiplier: multiplier,
		isWhiteTurn:          true,
		isWhiteUnderCheck:    false,
		isBlackUnderCheck:    false,
		whiteTimer:           int(match.FullTime),
		blackTimer:           int(match.FullTime),
		matchId:              match.ID,
	}

	cur := cfg.Matches[newGame]

	cfg.fillBoard(newGame)
	UpdateCoordinates(&cur)
	http.SetCookie(w, &startGame)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   match.White,
		Timer:  formatTime(int(match.FullTime)),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   match.Black,
		Timer:  formatTime(int(match.FullTime)),
		Pieces: "black",
	}

	moves, err := cfg.database.GetAllMovesForMatch(r.Context(), match.ID)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't get all moves", err)
		return
	}

	err = layout.MatchHistoryBoard(cur.board, cur.pieces, cur.coordinateMultiplier, whitePlayer, blackPlayer, cur.takenPiecesWhite, cur.takenPiecesBlack, moves).Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}

func (cfg *appConfig) moveHistoryHandler(w http.ResponseWriter, r *http.Request) {
	tile := r.PathValue("tile")
	c, err := r.Cookie("current_game")

	if err != nil {
		respondWithAnError(w, http.StatusNotFound, "no game found", err)
		return
	}

	cookie := strings.Split(c.Value, "-")

	matchId, err := strconv.Atoi(cookie[1])

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert value", err)
		return
	}

	board, err := cfg.database.GetBoardForMove(r.Context(), database.GetBoardForMoveParams{
		MatchID: int32(matchId),
		Move:    tile,
	})

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't get the board for the move", err)
		return
	}

	var boardState map[string]string

	err = json.Unmarshal(board.Board, &boardState)

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't unmarshal board state", err)
		return
	}

	startingPieces := MakePieces()

	pieces := make(map[string]components.Piece, 0)

	for k, v := range boardState {
		curr := startingPieces[k]
		curr.Tile = v
		pieces[k] = curr
	}

	cfg.cleanFillBoard(c.Value, pieces)

	curr := cfg.Matches[c.Value]

	err = components.UpdateBoardHistory(curr.board, pieces, curr.coordinateMultiplier, formatTime(int(board.WhiteTime)), formatTime(int(board.BlackTime))).Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't render template", err)
		return
	}
}
