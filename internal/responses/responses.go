package responses

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

func RespondWithAnError(w http.ResponseWriter, code int, message string, err error) {
	caller := getCaller()
	log.Printf("%v -> %v:%v\n", caller, message, err)
	w.WriteHeader(code)
}

func RespondWithAnErrorPage(w http.ResponseWriter, r *http.Request, code int, message string) {
	caller := getCaller()
	log.Printf("%v -> %v\n", caller, message)
	err := errorPage.Error(code, message).Render(r.Context(), w)
	if err != nil {
		RespondWithAnError(w, http.StatusInternalServerError, "Couldn't render template in render Page", err)
		return
	}
}

func LogError(message string, err error) {
	caller := getCaller()
	log.Printf("%v -> %v:%v\n", caller, message, err)
}

func RespondWithNewPiece(w http.ResponseWriter, r *http.Request, square components.Square) error {
	err := r.ParseForm()

	if err != nil {
		RespondWithAnError(w, http.StatusInternalServerError, "couldn't decode request", err)
		return err
	}

	multiplier, err := strconv.Atoi(r.FormValue("multiplier"))

	if err != nil {
		RespondWithAnError(w, http.StatusInternalServerError, "couldn't convert multiplier", err)
		return err
	}
	_, err = fmt.Fprintf(
		w,
		GetSinglePieceMessage(),
		square.Piece.Name,
		square.CoordinatePosition[0]*multiplier,
		square.CoordinatePosition[1]*multiplier,
		square.Piece.Image,
		"",
	)

	return err
}
