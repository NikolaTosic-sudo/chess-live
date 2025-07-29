package board

import "fmt"

var cols = [8]string{"a", "b", "c", "d", "e", "f", "g", "h"}
var rows = [8]int{8, 7, 6, 5, 4, 3, 2, 1}

type Square struct {
	Piece       string
	Selected    bool
	Coordinates [2]int
	Color       string
}

func generateColor(n, m int, selected bool) string {

	if selected {
		return "background-color: #add8e6"
	}

	if n%2 == 0 && m%2 == 1 {
		return "background-color: #b58863"
	}

	if n%2 == 0 && m%2 == 0 {
		return "background-color: #f0d9b5"
	}

	if n%2 == 1 && m%2 == 1 {
		return "background-color: #f0d9b5"
	}

	return "background-color: #b58863"
}

func genCol(color string) string {
	return "background-color: " + color
}

func getPosX(row int) string {
	return fmt.Sprintf("top: %vpx", row*100)
}

func getPosY(row int) string {
	return fmt.Sprintf("left: %vpx", row*100-15)
}

func getPiecePos(cord [2]int) string {
	return fmt.Sprintf("bottom: %vpx; left: %vpx", cord[0], cord[1])
}
