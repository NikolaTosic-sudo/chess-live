package main

import (
	"reflect"
	"testing"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/google/uuid"
)

func TestCanPlay(t *testing.T) {

	tests := []struct {
		name          string
		match         Match
		piece         components.Piece
		onlinePlayers map[string]components.OnlinePlayerStruct
		userId        uuid.UUID
		wantResult    bool
	}{
		{
			name:  "Online game with white player",
			match: Match{isWhiteTurn: true},
			piece: components.Piece{IsWhite: true},
			onlinePlayers: map[string]components.OnlinePlayerStruct{
				"white": {
					ID: uuid.MustParse("9e7d4b5a-4a3e-4b30-b7a7-84a5b9de4dc3"),
				},
			},
			userId:     uuid.MustParse("9e7d4b5a-4a3e-4b30-b7a7-84a5b9de4dc3"),
			wantResult: true,
		},
		{
			name:  "Online game with black player",
			match: Match{isWhiteTurn: false},
			piece: components.Piece{IsWhite: false},
			onlinePlayers: map[string]components.OnlinePlayerStruct{
				"black": {
					ID: uuid.MustParse("9e7d4b5a-4a3e-4b30-b7a7-84a5b9de4dc3"),
				},
			},
			userId:     uuid.MustParse("9e7d4b5a-4a3e-4b30-b7a7-84a5b9de4dc3"),
			wantResult: true,
		},
		{
			name:  "Online game with black player wrong user",
			match: Match{isWhiteTurn: false},
			piece: components.Piece{IsWhite: false},
			onlinePlayers: map[string]components.OnlinePlayerStruct{
				"black": {
					ID: uuid.MustParse("9e7d4b5a-4a3e-4b30-b7a7-84a5b9de4dc3"),
				},
			},
			userId:     uuid.MustParse("9e7d4b5a-4a8e-4b30-b7a7-84a5b9de4dc3"),
			wantResult: false,
		},
		{
			name:          "Local game with white player",
			match:         Match{isWhiteTurn: true},
			piece:         components.Piece{IsWhite: true},
			onlinePlayers: nil,
			wantResult:    true,
		},
		{
			name:          "Local game with black player",
			match:         Match{isWhiteTurn: false},
			piece:         components.Piece{IsWhite: false},
			onlinePlayers: nil,
			wantResult:    true,
		},
		{
			name:          "Local game can't play",
			match:         Match{isWhiteTurn: true},
			piece:         components.Piece{IsWhite: false},
			onlinePlayers: nil,
			wantResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canPlayResult := tt.match.canPlay(tt.piece, tt.onlinePlayers, tt.userId)

			if canPlayResult != tt.wantResult {
				t.Errorf("canPlay() canPlayResult = %v, want %v", canPlayResult, tt.wantResult)
			}
		})
	}
}

func TestCanEat(t *testing.T) {
	tests := []struct {
		name          string
		selectedPiece components.Piece
		currentPiece  components.Piece
		wantResult    bool
	}{
		{
			name:          "Can eat white piece",
			selectedPiece: components.Piece{IsWhite: true},
			currentPiece:  components.Piece{IsWhite: false},
			wantResult:    true,
		},
		{
			name:          "Can eat black piece",
			selectedPiece: components.Piece{IsWhite: false},
			currentPiece:  components.Piece{IsWhite: true},
			wantResult:    true,
		},
		{
			name:          "Can NOT eat white piece",
			selectedPiece: components.Piece{IsWhite: true},
			currentPiece:  components.Piece{IsWhite: true},
			wantResult:    false,
		},
		{
			name:          "Can NOT eat black piece",
			selectedPiece: components.Piece{IsWhite: true},
			currentPiece:  components.Piece{IsWhite: true},
			wantResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canEatResult := canEat(tt.selectedPiece, tt.currentPiece)

			if canEatResult != tt.wantResult {
				t.Errorf("canEat() canEatResult = %v, want %v", canEatResult, tt.wantResult)
			}
		})
	}
}

func TestReplaceStyles(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		bottom     []int
		left       []int
		wantResult string
	}{
		{
			name: "Replace styles for a single component",
			text: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 200px; left: 200px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
			bottom: []int{400},
			left:   []int{400},
			wantResult: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 400px; left: 400px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
		},
		{
			name: "Replace styles for 2 components",
			text: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 200px; left: 200px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>

				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 200px; left: 200px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
			bottom: []int{400, 600},
			left:   []int{400, 600},
			wantResult: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 400px; left: 400px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>

				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 600px; left: 600px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacedText := replaceStyles(tt.text, tt.bottom, tt.left)

			if replacedText != tt.wantResult {
				t.Errorf("replaceStyles() replacedText = %v, want %v", replacedText, tt.wantResult)
			}
		})
	}
}

func TestCheckForCastle(t *testing.T) {
	tests := []struct {
		name         string
		match        Match
		currentPiece components.Piece
		wantResult   bool
	}{
		{
			name:  "Yes to the white small castle",
			match: getMockMatchCastle(),
			currentPiece: components.Piece{
				Name:       "left_white_rook",
				Image:      "white_rook",
				Tile:       "1h",
				IsWhite:    true,
				LegalMoves: [][]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}},
				Moved:      false,
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCastle, _ := tt.match.checkForCastle(tt.currentPiece)

			if isCastle != tt.wantResult {
				t.Errorf("checkForCastle() isCastle = %v, want %v", isCastle, tt.wantResult)
			}
		})
	}
}

func TestCheckLegalMoves(t *testing.T) {
	tests := []struct {
		name       string
		match      Match
		wantResult []string
	}{
		{
			name:       "Yes to the white small castle",
			match:      getMockMatchMovesKnight(),
			wantResult: []string{"1g", "4h", "4d", "5g", "5e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			legalMoves := tt.match.checkLegalMoves()

			if !reflect.DeepEqual(legalMoves, tt.wantResult) {
				t.Errorf("checkLegalMoves() legalMoves = %v, want %v", legalMoves, tt.wantResult)
			}
		})
	}
}
