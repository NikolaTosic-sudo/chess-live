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
)

func (gcfg *gameConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	cfg := gcfg.Matches["initial"]
	gcfg.fillBoard("initial")
	// gcfg.checkUser(r)

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Guest",
		Timer:  formatTime(cfg.whiteTimer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Opponent",
		Timer:  formatTime(cfg.blackTimer),
		Pieces: "black",
	}

	err := layout.MainPage(cfg.board, cfg.pieces, cfg.coordinateMultiplier, whitePlayer, blackPlayer).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (gcfg *gameConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentPieceName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	cfg := gcfg.Matches[currentGame]
	currentPiece := cfg.pieces[currentPieceName]
	canPlay := gcfg.canPlay(currentPiece, currentGame)
	currentSquareName := currentPiece.Tile
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile
	selSq := cfg.board[selectedSquare]

	legalMoves := gcfg.checkLegalMoves(currentGame)

	if canEat(cfg.selectedPiece, currentPiece) && slices.Contains(legalMoves, currentSquareName) {
		var kingCheck bool
		if cfg.selectedPiece.IsKing {
			kingCheck = gcfg.handleChecksWhenKingMoves(currentSquareName, currentGame)
		} else if cfg.isWhiteTurn && cfg.isWhiteUnderCheck && !slices.Contains(cfg.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck && !slices.Contains(cfg.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var check bool
		if !cfg.selectedPiece.IsKing {
			check, _, _ = gcfg.handleCheckForCheck(currentSquareName, currentGame, cfg.selectedPiece)
		}

		if check || kingCheck {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			cfg.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			cfg.selectedPiece.Image,
		)
		delete(cfg.pieces, currentPieceName)
		cfg.selectedPiece.Tile = currentSquareName
		cfg.selectedPiece.Moved = true
		cfg.pieces[cfg.selectedPiece.Name] = cfg.selectedPiece
		currentSquare.Piece = cfg.selectedPiece
		selSq.Piece = components.Piece{}
		cfg.board[currentSquareName] = currentSquare
		cfg.board[selectedSquare] = selSq
		saveSelected := cfg.selectedPiece
		cfg.selectedPiece = components.Piece{}

		gcfg.Matches[currentGame] = cfg

		noCheck := handleIfCheck(w, gcfg, saveSelected, currentGame)
		if noCheck {
			var kingName string
			if cfg.isWhiteUnderCheck {
				kingName = "white_king"
			} else if cfg.isBlackUnderCheck {
				kingName = "black_king"
			} else {
				gcfg.endTurn(w, r, currentGame)
				return
			}

			cfg.isWhiteUnderCheck = false
			cfg.isBlackUnderCheck = false
			cfg.tilesUnderAttack = []string{}
			getKing := cfg.pieces[kingName]
			getKingSquare := cfg.board[getKing.Tile]

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
		gcfg.endTurn(w, r, currentGame)

		return
	}
	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName && samePiece(cfg.selectedPiece, currentPiece) {

		isCastle, kingCheck := gcfg.checkForCastle(cfg.board, cfg.selectedPiece, currentPiece, currentGame)

		if isCastle && !cfg.isBlackUnderCheck && !cfg.isWhiteUnderCheck && !kingCheck {

			err := gcfg.handleCastle(w, currentPiece, currentGame)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error with handling castle", err)
			}
			return
		}

		var kingsName string
		var className string
		if cfg.isWhiteTurn && cfg.isWhiteUnderCheck {
			kingsName = "white_king"
		} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck {
			kingsName = "black_king"
		}

		if kingsName != "" && strings.Contains(cfg.selectedPiece.Name, kingsName) {
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
			cfg.selectedPiece.Name,
			selSq.Coordinates[0],
			selSq.Coordinates[1],
			cfg.selectedPiece.Image,
			className,
		)

		if err != nil {
			fmt.Println(err)
		}

		cfg.selectedPiece = currentPiece
		gcfg.Matches[currentGame] = cfg
		return
	}

	if currentSquare.Selected {
		currentSquare.Selected = false
		isKing := cfg.selectedPiece.IsKing
		cfg.selectedPiece = components.Piece{}
		cfg.board[currentSquareName] = currentSquare
		var kingsName string
		var className string
		if cfg.isWhiteTurn && cfg.isWhiteUnderCheck {
			kingsName = "white_king"
		} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck {
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

		gcfg.Matches[currentGame] = cfg

		return
	} else {
		currentSquare.Selected = true
		cfg.selectedPiece = currentPiece
		cfg.board[currentSquareName] = currentSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300 " />
			</span>
		`, currentPieceName, currentSquare.Coordinates[0], currentSquare.Coordinates[1], currentPiece.Image)

		gcfg.Matches[currentGame] = cfg
		return
	}
}

func (gcfg *gameConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	cfg := gcfg.Matches[currentGame]
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile

	legalMoves := gcfg.checkLegalMoves(currentGame)

	var kingCheck bool
	if cfg.selectedPiece.IsKing && slices.Contains(legalMoves, currentSquareName) {
		kingCheck = gcfg.handleChecksWhenKingMoves(currentSquareName, currentGame)
	} else if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var check bool
	if !cfg.selectedPiece.IsKing {
		check, _, _ = gcfg.handleCheckForCheck(currentSquareName, currentGame, cfg.selectedPiece)
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
			cfg.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			cfg.selectedPiece.Image,
		)
		saveSelected := cfg.selectedPiece
		bigCleanup(currentSquareName, &cfg)

		noCheck := handleIfCheck(w, gcfg, saveSelected, currentGame)
		if noCheck {
			cfg.isWhiteUnderCheck = false
			cfg.isBlackUnderCheck = false
		}

		gcfg.Matches[currentGame] = cfg
		gcfg.endTurn(w, r, currentGame)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (gcfg *gameConfig) coverCheckHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	cfg := gcfg.Matches[currentGame]
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile

	legalMoves := gcfg.checkLegalMoves(currentGame)

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var check bool
	var kingCheck bool
	if cfg.selectedPiece.IsKing {
		kingCheck = gcfg.handleChecksWhenKingMoves(currentSquareName, currentGame)
	} else {
		check, _, _ = gcfg.handleCheckForCheck(currentSquareName, currentGame, cfg.selectedPiece)
	}
	if check || kingCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var kingName string

	if cfg.isWhiteTurn {
		kingName = "white_king"
	} else {
		kingName = "black_king"
	}

	king := cfg.pieces[kingName]
	kingSquare := cfg.board[king.Tile]

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
			cfg.selectedPiece.Name,
			currentSquare.Coordinates[0],
			currentSquare.Coordinates[1],
			cfg.selectedPiece.Image,
		)
		saveSelected := cfg.selectedPiece
		bigCleanup(currentSquareName, &cfg)

		for _, tile := range cfg.tilesUnderAttack {
			t := cfg.board[tile]
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

		noCheck := handleIfCheck(w, gcfg, saveSelected, currentGame)
		if noCheck {
			cfg.isWhiteUnderCheck = false
			cfg.isBlackUnderCheck = false
		}

		gcfg.Matches[currentGame] = cfg
		gcfg.endTurn(w, r, currentGame)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (gcfg *gameConfig) timerHandler(w http.ResponseWriter, r *http.Request) {

	c, err := r.Cookie("current_game")
	if err != nil {
		fmt.Println("no game found")
		return
	}
	currentGame := c.Value
	cfg := gcfg.Matches[currentGame]

	var toChangeColor string
	var stayTheSameColor string
	var toChange int
	var stayTheSame int

	if cfg.isWhiteTurn {
		toChangeColor = "white"
		cfg.whiteTimer -= 1
		toChange = cfg.whiteTimer
		stayTheSame = cfg.blackTimer
		stayTheSameColor = "black"
	} else {
		cfg.blackTimer -= 1
		toChangeColor = "black"
		toChange = cfg.blackTimer
		stayTheSame = cfg.whiteTimer
		stayTheSameColor = "white"
	}

	fmt.Fprintf(w, `	
				<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-white">%v</div>

				<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

			`, toChangeColor, formatTime(toChange), stayTheSameColor, formatTime(stayTheSame))

	gcfg.Matches[currentGame] = cfg
}

type MultiplerBody struct {
	Multiplier int `json:"multiplier"`
}

func (gcfg *gameConfig) updateMultiplerHandler(w http.ResponseWriter, r *http.Request) {

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
	cfg := gcfg.Matches[currentGame]

	cfg.coordinateMultiplier = multiplier
	UpdateCoordinates(&cfg)
	gcfg.Matches[currentGame] = cfg

	for k, piece := range cfg.pieces {
		tile := cfg.board[piece.Tile]

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile-md tile hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			k, tile.Coordinates[0], tile.Coordinates[1], piece.Image)
	}
}

func (gcfg *gameConfig) startGameHandler(w http.ResponseWriter, r *http.Request) {

	randomString, err := auth.MakeRefreshToken()

	if err != nil {
		fmt.Println(err)
		return
	}

	startGame := http.Cookie{
		Name:     "current_game",
		Value:    randomString,
		Path:     "/",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	startingBoard := MakeBoard()
	startingPieces := MakePieces()

	gcfg.Matches[randomString] = Match{
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

	cur := gcfg.Matches[randomString]

	gcfg.fillBoard(randomString)
	UpdateCoordinates(&cur)
	http.SetCookie(w, &startGame)

	fmt.Fprintf(w, `

		<div id="right-side"></div>

		<div id="overlay" hx-swap-oob="true" class="hidden w-board w-board-md h-board h-board-md absolute z-20 hover:cursor-default">
    </div>

		<div id="timer-update" hx-get="/timer" hx-trigger="every 1s" hx-swap-oob="true"></div>
	`)
}

func (cfg *apiConfig) timeOptionHandler(w http.ResponseWriter, r *http.Request) {

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

func (gcfg *gameConfig) setTimeOption(w http.ResponseWriter, r *http.Request) {

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
	cfg := gcfg.Matches[currentGame]

	var seconds string

	if a != 0 {
		seconds = fmt.Sprintf("+ %v sec", a)
		cfg.addition = a
	}

	cfg.whiteTimer = t * 60
	cfg.blackTimer = t * 60

	gcfg.Matches[currentGame] = cfg

	fmt.Fprintf(w, `
		<div id="dropdown-menu" hx-swap-oob="true" class="relative mb-8"></div>

		<div id="white" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		<div id="black" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		%v Min %v
	`, formatTime(cfg.whiteTimer), formatTime(cfg.blackTimer), time, seconds)
}

func (cfg *apiConfig) loginOpenHandler(w http.ResponseWriter, r *http.Request) {
	err := layout.LoginModal().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *apiConfig) closeModalHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{})
}

func (cfg *apiConfig) signupModalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := components.Signup().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *apiConfig) loginModalHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	err := components.Login().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (cfg *apiConfig) signupHandler(w http.ResponseWriter, r *http.Request) {
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

	http.SetCookie(w, &c)
	http.SetCookie(w, &refreshC)

	fmt.Fprintf(w, `
	<div id="modal-content" hx-swap-oob="innerHTML">
		<div class="flex justify-between items-center mb-6">
      <h2 class="text-xl font-semibold text-gray-100" id="modal-title">Welcome %v</h2>
      <button hx-get="/close-modal" class="text-gray-400 hover:text-gray-200 text-2xl leading-none">&times;</button>
    </div>
		<div class="text-gray-100">You signed up successfully</div>
	</div>
	`, user.Name)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
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

	http.SetCookie(w, &c)
	http.SetCookie(w, &refreshC)

	fmt.Fprintf(w, `
	<div id="modal-content" hx-swap-oob="innerHTML">
		<div class="flex justify-between items-center mb-6">
      <h2 class="text-xl font-semibold text-gray-100" id="modal-title">Welcome %v</h2>
      <button hx-get="/close-modal" class="text-gray-400 hover:text-gray-200 text-2xl leading-none">&times;</button>
    </div>
		<div class="text-gray-100">You signed up successfully</div>
	</div>
	`, user.Name)
}
