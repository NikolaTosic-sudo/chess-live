package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/containers/errorPage"
)

func getCaller() string {
	if _, file, line, ok := runtime.Caller(2); ok {
		filePaths := strings.Split(file, "/")
		return fmt.Sprintf("%v:%v", filePaths[len(filePaths)-1], line)
	}
	return ""
}

func respondWithAnError(w http.ResponseWriter, code int, message string, err error) {
	caller := getCaller()
	log.Printf("%v -> %v:%v\n", caller, message, err)
	w.WriteHeader(code)
}

func respondWithAnErrorPage(w http.ResponseWriter, r *http.Request, code int, message string) {
	caller := getCaller()
	log.Printf("%v -> %v\n", caller, message)
	err := errorPage.Error(code, message).Render(r.Context(), w)
	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "Couldn't render template in render Page", err)
		return
	}
}

func logError(message string, err error) {
	caller := getCaller()
	log.Printf("%v -> %v:%v\n", caller, message, err)
}

func respondWithNewPiece(w http.ResponseWriter, r *http.Request, square components.Square) error {
	err := r.ParseForm()

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return err
	}

	multiplier, err := strconv.Atoi(r.FormValue("multiplier"))

	if err != nil {
		respondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return err
	}
	_, err = fmt.Fprintf(
		w,
		getSinglePieceMessage(),
		square.Piece.Name,
		square.CoordinatePosition[0]*multiplier,
		square.CoordinatePosition[1]*multiplier,
		square.Piece.Image,
		"",
	)

	return err
}

func (m *Match) respondWithCheck(w http.ResponseWriter, square components.Square, king components.Piece) error {
	onlineGame, found := m.isOnlineMatch()
	className := `class="bg-red-400"`
	message := fmt.Sprintf(
		getSinglePieceMessage(),
		king.Name,
		square.Coordinates[0],
		square.Coordinates[1],
		king.Image,
		className,
	)

	err := sendMessage(onlineGame, found, w, message, [2][]int{
		{square.CoordinatePosition[0]},
		{square.CoordinatePosition[1]},
	})

	return err
}

func (m *Match) respondWithCoverCheck(w http.ResponseWriter, tile string, t components.Square) error {
	onlineGame, found := m.isOnlineMatch()
	message := fmt.Sprintf(
		getTileMessage(),
		tile,
		"cover-check",
		t.Color,
	)

	err := sendMessage(onlineGame, found, w, message, [2][]int{})

	return err
}
