package matches

import (
	"fmt"
	"net/http"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
	"github.com/NikolaTosic-sudo/chess-live/internal/responses"
	"github.com/NikolaTosic-sudo/chess-live/internal/utils"
	"github.com/google/uuid"
)

func (m *Match) RespondWithCheck(w http.ResponseWriter, square components.Square, king components.Piece) error {
	className := `class="bg-red-400"`
	message := fmt.Sprintf(
		responses.GetSinglePieceMessage(),
		king.Name,
		square.Coordinates[0],
		square.Coordinates[1],
		king.Image,
		className,
	)

	err := m.SendMessage(w, message, [2][]int{
		{square.CoordinatePosition[0]},
		{square.CoordinatePosition[1]},
	})

	return err
}

func (m *Match) RespondWithCoverCheck(w http.ResponseWriter, tile string, t components.Square) error {
	message := fmt.Sprintf(
		responses.GetTileMessage(),
		tile,
		"cover-check",
		t.Color,
	)

	err := m.SendMessage(w, message, [2][]int{})

	return err
}

func (m *Match) SendMessage(w http.ResponseWriter, msg string, args [2][]int) error {
	onlineGame, found := m.IsOnlineMatch()

	if found && len(args) > 0 {
		for _, onlinePlayer := range onlineGame.Players {
			var bottomCoordinates []int
			for _, coordinate := range args[0] {
				bottomCoordinates = append(bottomCoordinates, coordinate*onlinePlayer.Multiplier)
			}
			var leftCoordinates []int
			for _, coordinate := range args[1] {
				leftCoordinates = append(leftCoordinates, coordinate*onlinePlayer.Multiplier)
			}

			newMessage := utils.ReplaceStyles(msg, bottomCoordinates, leftCoordinates)
			onlineGame.PlayerMsg <- newMessage
			onlineGame.Player <- onlinePlayer
		}

	} else if found {
		onlineGame.Message <- msg
	} else {
		_, err := fmt.Fprint(w, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Match) CheckForPawnPromotion(pawnName string, w http.ResponseWriter, userId uuid.UUID) (bool, error) {
	var isOnLastTile bool
	onlineGame, found := m.IsOnlineMatch()
	pawn := m.Pieces[pawnName]
	if !pawn.IsPawn {
		return false, nil
	}
	square := m.Board[pawn.Tile]
	var pieceColor string
	var firstPosition string
	if pawn.IsWhite {
		pieceColor = "white"
		firstPosition = "top: 0px"
	} else {
		pieceColor = "black"
		firstPosition = "bottom: 0px"
	}

	var multiplier int

	if found {
		for _, player := range onlineGame.Players {
			if player.ID == userId {
				multiplier = player.Multiplier
			}
		}
	} else {
		multiplier = m.CoordinateMultiplier
	}
	endBoardCoordinates := 7 * multiplier
	dropdownPosition := square.Coordinates[1] + multiplier
	if square.Coordinates[1] == endBoardCoordinates {
		dropdownPosition = square.Coordinates[1] - multiplier
	}

	rowIdx := RowIdxMap[string(pawn.Tile[0])]
	if pawn.IsWhite && rowIdx == 0 || !pawn.IsWhite && rowIdx == 7 {
		isOnLastTile = true
		message := fmt.Sprintf(
			responses.GetPromotionInitMessage(),
			firstPosition,
			dropdownPosition,
			pieceColor,
			pawnName,
			pieceColor,
			pieceColor,
			pawnName,
			pieceColor,
			pieceColor,
			pawnName,
			pieceColor,
			pieceColor,
			pawnName,
			pieceColor,
		)

		_, err := fmt.Fprint(w, message)

		if err != nil {
			return false, err
		}
	}
	return isOnLastTile, nil
}
