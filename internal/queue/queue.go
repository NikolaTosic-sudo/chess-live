package queue

import (
	"fmt"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
)

type State string

const (
	StateOpen State = "Open"
	StateDone State = "Done"
)

type PlayersQueue struct {
	players []components.OnlinePlayerStruct
	state   State
}

func (q *PlayersQueue) NewQueue() PlayersQueue {
	queue := PlayersQueue{
		players: []components.OnlinePlayerStruct{},
		state:   StateOpen,
	}
	return queue
}

func (q *PlayersQueue) Enqueue(player components.OnlinePlayerStruct) {
	if q.isFull() || q.IsDone() {
		return
	}

	q.players = append(q.players, player)
}

func (q *PlayersQueue) Dequeue() (components.OnlinePlayerStruct, error) {
	if q.isEmpty() {
		return components.OnlinePlayerStruct{}, fmt.Errorf("tried dequeue on an empty queue")
	}
	player := q.players[0]
	if q.isFull() {
		q.players = q.players[1:]
	} else {
		q.players = []components.OnlinePlayerStruct{}
		q.Done()
	}
	return player, nil
}

func (q *PlayersQueue) Done() {
	q.state = StateDone
}

func (q *PlayersQueue) isFull() bool {
	return len(q.players) == 2
}

func (q *PlayersQueue) isEmpty() bool {
	return len(q.players) == 0
}

func (q *PlayersQueue) IsDone() bool {
	return q.state == StateDone
}

func (q *PlayersQueue) hasItems() bool {
	return len(q.players) > 0
}

func (q *PlayersQueue) HasSpot() bool {
	return !q.isFull() && q.hasItems() && !q.isEmpty() && !q.IsDone()
}
