package main

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/NikolaTosic-sudo/chess-live/components/board"
)

func (cfg *apiConfig) boardHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fillBoard()
	err := board.GridBoard(cfg.board, cfg.pieces).Render(r.Context(), w)

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
		if cfg.isWhiteTurn && cfg.isWhiteUnderCheck && !slices.Contains(cfg.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck && !slices.Contains(cfg.tilesUnderAttack, currentSquareName) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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
		selSq.Piece = board.Piece{}
		cfg.board[currentSquareName] = currentSquare
		cfg.board[selectedSquare] = selSq
		saveSelected := cfg.selectedPiece
		cfg.selectedPiece = board.Piece{}
		cfg.isWhiteTurn = !cfg.isWhiteTurn

		check, king, tilesUnderAttack := cfg.handleCheckForCheck("", saveSelected)
		kingSquare := cfg.board[king.Tile]

		if check {
			if king.IsWhite {
				cfg.isWhiteUnderCheck = true
			} else {
				cfg.isBlackUnderCheck = true
			}
			_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-red-400 " />
			</span>
		`,
				king.Name,
				kingSquare.Coordinates[0],
				kingSquare.Coordinates[1],
				king.Image,
			)

			if err != nil {
				fmt.Println(err)
			}

			cfg.tilesUnderAttack = tilesUnderAttack

			for _, tile := range tilesUnderAttack {
				t := cfg.board[tile]

				if t.Piece.Name != "" {
					_, err := fmt.Fprintf(w, `
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
						<img src="/assets/pieces/%v.svg" />
					</span>
				`,
						t.Piece.Name,
						t.Coordinates[0],
						t.Coordinates[1],
						t.Piece.Image,
					)

					if err != nil {
						fmt.Println(err)
					}
				} else {
					_, err := fmt.Fprintf(w, `
						<div id="%v" hx-post="/cover-check" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>
				`,
						tile,
						t.Color,
					)

					if err != nil {
						fmt.Println(err)
					}
				}
			}
		} else {
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
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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

		isCastle := checkForCastle(cfg.board, cfg.selectedPiece, currentPiece)

		if isCastle {
			err := cfg.handleCastle(w, currentPiece)
			if err != nil {
				respondWithAnError(w, http.StatusInternalServerError, "error with handling castle", err)
			}
			return
		}

		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-sky-300 " />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
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
		)
		cfg.selectedPiece = currentPiece
		return
	}

	if currentSquare.Selected {
		currentSquare.Selected = false
		cfg.selectedPiece = board.Piece{}
		cfg.board[currentSquareName] = currentSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>
		`, currentPieceName, currentSquare.Coordinates[0], currentSquare.Coordinates[1], currentPiece.Image)
		return
	} else {
		currentSquare.Selected = true
		cfg.selectedPiece = currentPiece
		cfg.board[currentSquareName] = currentSquare
		fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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
	selSeq := cfg.board[selectedSquare]

	if cfg.isWhiteTurn && cfg.isWhiteUnderCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	} else if !cfg.isWhiteTurn && cfg.isBlackUnderCheck {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	legalMoves := cfg.checkLegalMoves()

	if !slices.Contains(legalMoves, currentSquareName) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	check, _, _ := cfg.handleCheckForCheck(currentSquareName, cfg.selectedPiece)

	if check {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if selectedSquare != "" && selectedSquare != currentSquareName {
		fmt.Fprintf(w, `
			<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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
		currentSquare.Selected = false
		currentPiece := cfg.pieces[cfg.selectedPiece.Name]
		currentPiece.Tile = currentSquareName
		currentPiece.Moved = true
		cfg.pieces[cfg.selectedPiece.Name] = currentPiece
		currentSquare.Piece = currentPiece
		saveSelected := cfg.selectedPiece
		cfg.selectedPiece = board.Piece{}
		selSeq.Piece = cfg.selectedPiece
		cfg.board[selectedSquare] = selSeq
		cfg.board[currentSquareName] = currentSquare

		check, king, tilesUnderAttack := cfg.handleCheckForCheck("", saveSelected)
		kingSquare := cfg.board[king.Tile]

		if check {
			if king.IsWhite {
				cfg.isWhiteUnderCheck = true
			} else {
				cfg.isBlackUnderCheck = true
			}
			_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-red-400 " />
			</span>
		`,
				king.Name,
				kingSquare.Coordinates[0],
				kingSquare.Coordinates[1],
				king.Image,
			)

			if err != nil {
				fmt.Println(err)
			}

			cfg.tilesUnderAttack = tilesUnderAttack

			for _, tile := range tilesUnderAttack {
				t := cfg.board[tile]

				if t.Piece.Name != "" {
					_, err := fmt.Fprintf(w, `
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
						<img src="/assets/pieces/%v.svg" />
					</span>
				`,
						t.Piece.Name,
						t.Coordinates[0],
						t.Coordinates[1],
						t.Piece.Image,
					)

					if err != nil {
						fmt.Println(err)
					}
				} else {
					_, err := fmt.Fprintf(w, `
						<div id="%v" hx-post="/cover-check" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>
				`,
						tile,
						t.Color,
					)

					if err != nil {
						fmt.Println(err)
					}
				}
			}
		}

		cfg.isWhiteTurn = !cfg.isWhiteTurn

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) coverCheckHandler(w http.ResponseWriter, r *http.Request) {
	currentSquareName := r.Header.Get("Hx-Trigger")
	currentSquare := cfg.board[currentSquareName]
	selectedSquare := cfg.selectedPiece.Tile
	selSeq := cfg.board[selectedSquare]

	legalMoves := cfg.checkLegalMoves()

	if !slices.Contains(legalMoves, currentSquareName) {
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
			<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" />
			</span>

			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
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
		currentSquare.Selected = false
		currentPiece := cfg.pieces[cfg.selectedPiece.Name]
		currentPiece.Tile = currentSquareName
		currentPiece.Moved = true
		cfg.pieces[cfg.selectedPiece.Name] = currentPiece
		currentSquare.Piece = currentPiece
		saveSelected := cfg.selectedPiece
		cfg.selectedPiece = board.Piece{}
		selSeq.Piece = cfg.selectedPiece
		cfg.board[selectedSquare] = selSeq
		cfg.board[currentSquareName] = currentSquare

		for _, tile := range cfg.tilesUnderAttack {
			t := cfg.board[tile]
			if t.Piece.Name != "" {
				_, err := fmt.Fprintf(w, `
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
						<img src="/assets/pieces/%v.svg" />
					</span>
				`,
					t.Piece.Name,
					t.Coordinates[0],
					t.Coordinates[1],
					t.Piece.Image,
				)

				if err != nil {
					fmt.Println(err)
				}
			} else {
				_, err := fmt.Fprintf(w, `
						<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>
				`,
					tile,
					t.Color,
				)

				if err != nil {
					fmt.Println(err)
				}
			}
		}

		check, king, tilesUnderAttack := cfg.handleCheckForCheck("", saveSelected)
		kingSquare := cfg.board[king.Tile]

		if check {
			if king.IsWhite {
				cfg.isWhiteUnderCheck = true
			} else {
				cfg.isBlackUnderCheck = true
			}
			_, err := fmt.Fprintf(w, `
			<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
				<img src="/assets/pieces/%v.svg" class="bg-red-400 " />
			</span>
		`,
				king.Name,
				kingSquare.Coordinates[0],
				kingSquare.Coordinates[1],
				king.Image,
			)

			if err != nil {
				fmt.Println(err)
			}

			cfg.tilesUnderAttack = tilesUnderAttack

			for _, tile := range tilesUnderAttack {
				t := cfg.board[tile]

				if t.Piece.Name != "" {
					_, err := fmt.Fprintf(w, `
					<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="w-[100px] h-[100px] hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
						<img src="/assets/pieces/%v.svg" />
					</span>
				`,
						t.Piece.Name,
						t.Coordinates[0],
						t.Coordinates[1],
						t.Piece.Image,
					)

					if err != nil {
						fmt.Println(err)
					}
				} else {
					_, err := fmt.Fprintf(w, `
						<div id="%v" hx-post="/cover-check" hx-swap-oob="true" class="max-w-[100px] max-h-[100px] h-full w-full" style="background-color: %v"></div>
				`,
						tile,
						t.Color,
					)

					if err != nil {
						fmt.Println(err)
					}
				}
			}
		}

		cfg.isWhiteTurn = !cfg.isWhiteTurn

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
