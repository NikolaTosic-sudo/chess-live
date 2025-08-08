package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
	"github.com/NikolaTosic-sudo/chess-live/internal/auth"
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/google/uuid"
)

func (cfg *appConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	match := cfg.Matches["initial"]
	cfg.fillBoard("initial")

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

	err := layout.MainPage(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) privateBoardHandler(w http.ResponseWriter, r *http.Request) {
	var userName string
	userC, err := r.Cookie("access_token")

	if err != nil {
		fmt.Println(err)
		userName = "Guest"
	} else if userC.Value != "" {
		userId, err := auth.ValidateJWT(userC.Value, cfg.secret)

		if err != nil {
			fmt.Println(err)
			userName = "Guest"
		}

		user, err := cfg.database.GetUserById(r.Context(), userId)

		if err != nil {
			fmt.Println(err)
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
		fmt.Println(err)
		game = "initial"
	} else if c.Value != "" {
		game = c.Value
	} else {
		game = "initial"
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

	err = layout.MainPagePrivate(match.board, match.pieces, match.coordinateMultiplier, whitePlayer, blackPlayer).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *appConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentPieceName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	match := cfg.Matches[currentGame]
	currentPiece := match.pieces[currentPieceName]
	canPlay := cfg.canPlay(currentPiece, currentGame)
	currentSquareName := currentPiece.Tile
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile
	selSq := match.board[selectedSquare]

	legalMoves := cfg.checkLegalMoves(currentGame)

	if canEat(match.selectedPiece, currentPiece) && slices.Contains(legalMoves, currentSquareName) {
		var kingCheck bool
		if match.selectedPiece.IsKing {
			kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName, currentGame)
		} else if match.isWhiteTurn && match.isWhiteUnderCheck && !slices.Contains(match.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		} else if !match.isWhiteTurn && match.isBlackUnderCheck && !slices.Contains(match.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var check bool
		if !match.selectedPiece.IsKing {
			check, _, _ = cfg.handleCheckForCheck(currentSquareName, currentGame, match.selectedPiece)
		}

		if check || kingCheck {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var userColor string
		if match.isWhiteTurn {
			userColor = "white"
		} else {
			userColor = "black"
		}

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<div id="lost-pieces-%v" hx-swap-oob="afterbegin">
				<img src="/assets/pieces/%v.svg" class="w-[18px] h-[18px]" />
			</div>
		`,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
			userColor,
			currentPiece.Image,
		)
		delete(match.pieces, currentPieceName)
		match.selectedPiece.Tile = currentSquareName
		match.selectedPiece.Moved = true
		match.pieces[match.selectedPiece.Name] = match.selectedPiece
		currentSquare.Piece = match.selectedPiece
		selSq.Piece = components.Piece{}
		match.board[currentSquareName] = currentSquare
		match.board[selectedSquare] = selSq
		saveSelected := match.selectedPiece
		match.selectedPiece = components.Piece{}

		cfg.Matches[currentGame] = match

		noCheck := handleIfCheck(w, cfg, saveSelected, currentGame)
		if noCheck {
			var kingName string
			if match.isWhiteUnderCheck {
				kingName = "white_king"
			} else if match.isBlackUnderCheck {
				kingName = "black_king"
			} else {
				cfg.endTurn(w, r, currentGame)
				return
			}

			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
			match.tilesUnderAttack = []string{}
			getKing := match.pieces[kingName]
			getKingSquare := match.board[getKing.Tile]

			fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
				getKing.Name,
				getKingSquare.Coordinates[0],
				getKingSquare.Coordinates[1],
				getKing.Image,
			)
		}
		cfg.endTurn(w, r, currentGame)

		return
	}
	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName && samePiece(match.selectedPiece, currentPiece) {

		isCastle, kingCheck := cfg.checkForCastle(match.board, match.selectedPiece, currentPiece, currentGame)

		if isCastle && !match.isBlackUnderCheck && !match.isWhiteUnderCheck && !kingCheck {

			err := cfg.handleCastle(w, currentPiece, currentGame)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error with handling castle", err)
			}
			return
		}

		var kingsName string
		var className string
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			kingsName = "white_king"
		} else if !match.isWhiteTurn && match.isBlackUnderCheck {
			kingsName = "black_king"
		}

		if kingsName != "" && strings.Contains(match.selectedPiece.Name, kingsName) {
			className = `class="bg-red-400"`
		}

		_, err := fmt.Fprintf(w, `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
					<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
				</span>
	
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
					<img src="/assets/pieces/%v.svg" %v  />
				</span>
			`,
			currentPieceName,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			currentPiece.Image,
			match.selectedPiece.Name,
			selSq.Coordinates[0],
			selSq.Coordinates[1],
			match.selectedPiece.Image,
			className,
		)

		if err != nil {
			fmt.Println(err)
		}

		match.selectedPiece = currentPiece
		cfg.Matches[currentGame] = match
		return
	}

	if currentSquare.Selected {
		currentSquare.Selected = false
		isKing := match.selectedPiece.IsKing
		match.selectedPiece = components.Piece{}
		match.board[currentSquareName] = currentSquare
		var kingsName string
		var className string
		if match.isWhiteTurn && match.isWhiteUnderCheck {
			kingsName = "white_king"
		} else if !match.isWhiteTurn && match.isBlackUnderCheck {
			kingsName = "black_king"
		}
		if kingsName != "" && isKing {
			className = `class="bg-red-400"`
		}
		_, err := fmt.Fprintf(w, `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
			currentPieceName,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			currentPiece.Image,
			className,
		)

		if err != nil {
			fmt.Println(err)
		}

		cfg.Matches[currentGame] = match

		return
	} else {
		currentSquare.Selected = true
		match.selectedPiece = currentPiece
		match.board[currentSquareName] = currentSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300 " />
			</span>
		`, currentPieceName, currentSquare.Coordinates[0], currentSquare.Coordinates[1], currentPiece.Image)

		cfg.Matches[currentGame] = match
		return
	}
}

func (cfg *appConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	match := cfg.Matches[currentGame]
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile

	legalMoves := cfg.checkLegalMoves(currentGame)

	var kingCheck bool
	if match.selectedPiece.IsKing && slices.Contains(legalMoves, currentSquareName) {
		kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName, currentGame)
	} else if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var check bool
	if !match.selectedPiece.IsKing {
		check, _, _ = cfg.handleCheckForCheck(currentSquareName, currentGame, match.selectedPiece)
	}

	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		fmt.Fprintf(w, `
			<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="tile tile-md" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			currentSquareName,
			currentSquare.Color,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
		)
		saveSelected := match.selectedPiece
		match.allMoves = append(match.allMoves, currentSquareName)
		if len(match.allMoves)%2 == 0 {
			fmt.Fprintf(w, `
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v</span>
				</div>
			`,
				currentSquareName,
			)
		} else {
			fmt.Fprintf(w, `
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v.</span>
					<span>%v</span>
				</div>
		`,
				len(match.allMoves)/2+1,
				currentSquareName,
			)
		}
		bigCleanup(currentSquareName, &match)

		noCheck := handleIfCheck(w, cfg, saveSelected, currentGame)
		if noCheck {
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
		}

		cfg.Matches[currentGame] = match
		cfg.endTurn(w, r, currentGame)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) coverCheckHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	match := cfg.Matches[currentGame]
	currentSquare := match.board[currentSquareName]
	selectedSquare := match.selectedPiece.Tile

	legalMoves := cfg.checkLegalMoves(currentGame)

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var check bool
	var kingCheck bool
	if match.selectedPiece.IsKing {
		kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName, currentGame)
	} else {
		check, _, _ = cfg.handleCheckForCheck(currentSquareName, currentGame, match.selectedPiece)
	}
	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var kingName string

	if match.isWhiteTurn {
		kingName = "white_king"
	} else {
		kingName = "black_king"
	}

	king := match.pieces[kingName]
	kingSquare := match.board[king.Tile]

	if selectedSquare != "" && selectedSquare != currentSquareName {
		fmt.Fprintf(w, `
			<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="tile tile-md h-full w-full" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			currentSquareName,
			currentSquare.Color,
			king.Name,
			kingSquare.Coordinates[0],
			kingSquare.Coordinates[1],
			king.Image,
			match.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			match.selectedPiece.Image,
		)
		saveSelected := match.selectedPiece
		bigCleanup(currentSquareName, &match)

		for _, tile := range match.tilesUnderAttack {
			t := match.board[tile]
			if t.Piece.Name != "" {
				err := respondWithNewPiece(w, t)

				if err != nil {
					fmt.Println(err)
				}
			} else {
				_, err := fmt.Fprintf(w, `
						<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="tile tile-md" style="background-color: %v"></div>
				`,
					tile,
					t.Color,
				)

				if err != nil {
					fmt.Println(err)
				}
			}
		}

		noCheck := handleIfCheck(w, cfg, saveSelected, currentGame)
		if noCheck {
			match.isWhiteUnderCheck = false
			match.isBlackUnderCheck = false
		}

		cfg.Matches[currentGame] = match
		cfg.endTurn(w, r, currentGame)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *appConfig) timerHandler(w http.ResponseWriter, r *http.Request) {

	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	match := cfg.Matches[currentGame]

	var toChangeColor string
	var stayTheSameColor string
	var toChange int
	var stayTheSame int

	if match.isWhiteTurn {
		toChangeColor = "white"
		match.whiteTimer -= 1
		toChange = match.whiteTimer
		stayTheSame = match.blackTimer
		stayTheSameColor = "black"
	} else {
		match.blackTimer -= 1
		toChangeColor = "black"
		toChange = match.blackTimer
		stayTheSame = match.whiteTimer
		stayTheSameColor = "white"
	}

	fmt.Fprintf(w, `	
				<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-white">%v</div>

				<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

			`, toChangeColor, formatTime(toChange), stayTheSameColor, formatTime(stayTheSame))

	cfg.Matches[currentGame] = match
}

type MultiplerBody struct {
	Multiplier int `json:"multiplier"`
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
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	match := cfg.Matches[currentGame]

	match.coordinateMultiplier = multiplier
	UpdateCoordinates(&match)
	cfg.Matches[currentGame] = match

	for k, piece := range match.pieces {
		tile := match.board[piece.Tile]

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile-md tile hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			k, tile.Coordinates[0], tile.Coordinates[1], piece.Image)
	}
}

func (cfg *appConfig) startGameHandler(w http.ResponseWriter, r *http.Request) {

	user := cfg.isUserLoggedIn(r)

	var newGameName string

	if user != uuid.Nil {
		newGameName = user.String()
	} else {
		randomString, err := auth.MakeRefreshToken()

		if err != nil {
			fmt.Println(err)
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

	cfg.Matches[newGameName] = Match{
		board:                startingBoard,
		pieces:               startingPieces,
		selectedPiece:        components.Piece{},
		coordinateMultiplier: 80,
		isWhiteTurn:          true,
		isWhiteUnderCheck:    false,
		isBlackUnderCheck:    false,
		whiteTimer:           600,
		blackTimer:           600,
		addition:             0,
	}

	cur := cfg.Matches[newGameName]

	cfg.fillBoard(newGameName)
	UpdateCoordinates(&cur)
	http.SetCookie(w, &startGame)

	components.StartGameRight().Render(r.Context(), w)
}

func (cfg *appConfig) resumeGameHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")

	if err != nil {
		fmt.Println("no cookie found")
		w.WriteHeader(http.StatusNoContent)
		return
	} else if c.Value == "" {
		fmt.Println("no cookie value")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	match, ok := cfg.Matches[c.Value]

	if !ok {
		fmt.Println("game not found")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cfg.fillBoard(c.Value)
	UpdateCoordinates(&match)

	components.StartGameRight().Render(r.Context(), w)
}

func (cfg *appConfig) getAllMovesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("current_game")

	if err != nil {
		fmt.Println("no cookie found")
		w.WriteHeader(http.StatusNoContent)
		return
	} else if c.Value == "" {
		fmt.Println("no cookie value")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	match, ok := cfg.Matches[c.Value]

	if !ok {
		fmt.Println("game not found")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for i := 1; i <= len(match.allMoves); i++ {
		if i%2 == 0 {
			fmt.Fprintf(w, `
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v</span>
				</div>
			`,
				match.allMoves[i-1],
			)
		} else {
			fmt.Fprintf(w, `
				<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 text-white h-moves mt-8">
					<span>%v.</span>
					<span>%v</span>
				</div>
		`,
				i/2+1,
				match.allMoves[i-1],
			)
		}
	}
}

func (cfg *appConfig) timeOptionHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, `
		<div class="absolute right-0 mt-2 w-48 bg-[#1e1c1a] border border-[#3a3733] text-white rounded-md shadow-lg z-50">
			<div hx-post="/set-time" hx-vals='{"time": "15"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition">15 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "15", "addition": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition">15 + 3</div>
			<div hx-post="/set-time" hx-vals='{"time": "10"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition">10 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "10", "addition": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition">10 + 3</div>
			<div hx-post="/set-time" hx-vals='{"time": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition">3 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "3", "addition": "1"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition">3 + 1</div>
		</div>
	`)
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

	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	match := cfg.Matches[currentGame]

	var seconds string

	if a != 0 {
		seconds = fmt.Sprintf("+ %v sec", a)
		match.addition = a
	}

	match.whiteTimer = t * 60
	match.blackTimer = t * 60

	cfg.Matches[currentGame] = match

	fmt.Fprintf(w, `
		<div id="dropdown-menu" hx-swap-oob="true" class="relative mb-8"></div>

		<div id="white" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		<div id="black" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		%v Min %v
	`, formatTime(match.whiteTimer), formatTime(match.blackTimer), time, seconds)
}

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
				<div id="password" hx-swap-oob="afterend">
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

	http.SetCookie(w, &c)
	http.SetCookie(w, &refreshC)
	http.SetCookie(w, &cGC)

	cfg.users[user.ID] = CurrentUser{
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
				<div id="password" hx-swap-oob="afterend">
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
				<div id="password" hx-swap-oob="afterend">
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

	http.SetCookie(w, &c)
	http.SetCookie(w, &refreshC)
	http.SetCookie(w, &cGC)

	cfg.users[user.ID] = CurrentUser{
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

	http.SetCookie(w, &accC)
	http.SetCookie(w, &refreshC)
	http.SetCookie(w, &cGC)

	w.Header().Add("Hx-Redirect", "/")
}
