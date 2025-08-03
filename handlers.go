package main

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	layout "github.com/NikolaTosic-sudo/chess-live/containers/layouts"
)

func (cfg *apiConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fillBoard()

	timer := 600

	whitePlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Nikola",
		Timer:  formatTime(timer),
		Pieces: "white",
	}
	blackPlayer := components.PlayerStruct{
		Image:  "/assets/images/user-icon.png",
		Name:   "Ilma",
		Timer:  formatTime(timer),
		Pieces: "black",
	}
	err := layout.MainPage(cfg.board, cfg.pieces, cfg.coordinateMultiplier, whitePlayer, blackPlayer).Render(r.Context(), w)

	if err != nil {
		fmt.Println(err)
		respondWithAnErrorPage(w, r, http.StatusInternalServerError, "Couldn't render template")
		return
	}
}

func (cfg *apiConfig) moveHandler(w http.ResponseWriter, r *http.Request) {
	currentPieceName := r.Header.Get("Hx-Trigger")
	currentPiece := cfg.pieces[currentPieceName]
	canPlay := cfg.canPlay(currentPiece)
	currentSquareName := currentPiece.Tile
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile
	selSq := cfg.board[selectedSquare]

	legalMoves := cfg.checkLegalMoves()

	if canEat(cfg.selectedPiece, currentPiece) && slices.Contains(legalMoves, currentSquareName) {
		var kingCheck bool
		if cfg.selectedPiece.IsKing {
			kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName)
		} else if cfg.isWhiteTurn && cfg.isWhiteUnderCheck && !slices.Contains(cfg.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck && !slices.Contains(cfg.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var check bool
		if !cfg.selectedPiece.IsKing {
			check, _, _ = cfg.handleCheckForCheck(currentSquareName, cfg.selectedPiece)
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

		cfg.isWhiteTurn = !cfg.isWhiteTurn
		go cfg.gameDone()

		noCheck := handleIfCheck(w, cfg, saveSelected)
		if noCheck {
			var kingName string
			if cfg.isWhiteUnderCheck {
				kingName = "white_king"
			} else if cfg.isBlackUnderCheck {
				kingName = "black_king"
			} else {
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

		return
	}
	if !canPlay {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName && samePiece(cfg.selectedPiece, currentPiece) {

		isCastle, kingCheck := cfg.checkForCastle(cfg.board, cfg.selectedPiece, currentPiece)

		if isCastle && !cfg.isBlackUnderCheck && !cfg.isWhiteUnderCheck && !kingCheck {

			err := cfg.handleCastle(w, currentPiece)
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
		return
	}
}

func (cfg *apiConfig) moveToHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile

	legalMoves := cfg.checkLegalMoves()

	var kingCheck bool
	if cfg.selectedPiece.IsKing && slices.Contains(legalMoves, currentSquareName) {
		kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName)
	} else if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var check bool
	if !cfg.selectedPiece.IsKing {
		check, _, _ = cfg.handleCheckForCheck(currentSquareName, cfg.selectedPiece)
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
		bigCleanup(currentSquareName, cfg)

		noCheck := handleIfCheck(w, cfg, saveSelected)
		if noCheck {
			cfg.isWhiteUnderCheck = false
			cfg.isBlackUnderCheck = false
		}

		cfg.isWhiteTurn = !cfg.isWhiteTurn
		go cfg.gameDone()

		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) coverCheckHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile

	legalMoves := cfg.checkLegalMoves()

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var check bool
	var kingCheck bool
	if cfg.selectedPiece.IsKing {
		kingCheck = cfg.handleChecksWhenKingMoves(currentSquareName)
	} else {
		check, _, _ = cfg.handleCheckForCheck(currentSquareName, cfg.selectedPiece)
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
		bigCleanup(currentSquareName, cfg)

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

		noCheck := handleIfCheck(w, cfg, saveSelected)
		if noCheck {
			cfg.isWhiteUnderCheck = false
			cfg.isBlackUnderCheck = false
		}

		cfg.isWhiteTurn = !cfg.isWhiteTurn
		go cfg.gameDone()

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) timerHandler(w http.ResponseWriter, r *http.Request) {

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
				<div id="timer-update" hx-get="/timer" hx-trigger="every 1s" hx-swap-oob="none"></div>
	
				<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-amber-200">%v</div>

				<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-amber-200">%v</div>

			`, toChangeColor, formatTime(toChange), stayTheSameColor, formatTime(stayTheSame))
}

type MultiplerBody struct {
	Multiplier int `json:"multiplier"`
}

func (cfg *apiConfig) updateMultiplerHandler(w http.ResponseWriter, r *http.Request) {

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

	cfg.coordinateMultiplier = multiplier
	UpdateCoordinates(cfg)

	for k, piece := range cfg.pieces {
		tile := cfg.board[piece.Tile]

		fmt.Println(tile.Coordinates[0])

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile-md tile hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`,
			k, tile.Coordinates[0], tile.Coordinates[1], piece.Image)
	}
}
