package queue

import (
	"fmt"
	"testing"

	"github.com/NikolaTosic-sudo/chess-live/containers/components"
)

func TestEnqueue(t *testing.T) {
	tests := []struct {
		name             string
		queue            PlayersQueue
		noOfMethodsToRun int
		wantIsDone       bool
		wantIsFull       bool
		wantIsEmpty      bool
		wantHasSpot      bool
		wantLen          int
	}{
		{
			name:             "Add one to the empty queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 1,
			wantIsDone:       false,
			wantIsEmpty:      false,
			wantIsFull:       false,
			wantHasSpot:      true,
			wantLen:          1,
		},
		{
			name:             "Add 2 to the empty queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 2,
			wantIsDone:       false,
			wantIsEmpty:      false,
			wantIsFull:       true,
			wantHasSpot:      false,
			wantLen:          2,
		},
		{
			name:             "Add 3 to the empty queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 3,
			wantIsDone:       false,
			wantIsEmpty:      false,
			wantIsFull:       true,
			wantHasSpot:      false,
			wantLen:          2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.queue.NewQueue()

			for i := range tt.noOfMethodsToRun {
				q.Enqueue(components.OnlinePlayerStruct{
					Name: fmt.Sprintf("new game: %v", i),
				})
			}

			if q.IsDone() != tt.wantIsDone {
				t.Errorf("IsDone() = %v, want %v", q.IsDone(), tt.wantIsDone)
			}

			if q.isEmpty() != tt.wantIsEmpty {
				t.Errorf("IsEmpty = %v, want %v", q.isEmpty(), tt.wantIsEmpty)
			}

			if q.isFull() != tt.wantIsFull {
				t.Errorf("isFull = %v, want %v", q.isFull(), tt.wantIsFull)
			}

			if q.HasSpot() != tt.wantHasSpot {
				t.Errorf("HasSpot = %v, want %v", q.HasSpot(), tt.wantHasSpot)
			}

			if len(q.players) != tt.wantLen {
				t.Errorf("Length of players in queue = %v, want %v", len(q.players), tt.wantLen)
			}
		})
	}
}

func TestDequeu(t *testing.T) {
	tests := []struct {
		name             string
		queue            PlayersQueue
		noOfMethodsToRun int
		queueLen         int
		wantIsDone       bool
		wantIsFull       bool
		wantIsEmpty      bool
		wantHasSpot      bool
		wantLen          int
		wantErr          bool
	}{
		{
			name:             "Try dequeing from an empty queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 1,
			queueLen:         0,
			wantIsDone:       false,
			wantIsEmpty:      true,
			wantIsFull:       false,
			wantHasSpot:      false,
			wantLen:          0,
			wantErr:          true,
		},
		{
			name:             "Dequeue from a queue of length 1",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 1,
			queueLen:         1,
			wantIsDone:       true,
			wantIsEmpty:      true,
			wantIsFull:       false,
			wantHasSpot:      false,
			wantLen:          0,
			wantErr:          false,
		},
		{
			name:             "Dequeue once from a full queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 1,
			queueLen:         2,
			wantIsDone:       false,
			wantIsEmpty:      false,
			wantIsFull:       false,
			wantHasSpot:      true,
			wantLen:          1,
			wantErr:          false,
		},
		{
			name:             "Dequeue full queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 2,
			queueLen:         2,
			wantIsDone:       true,
			wantIsEmpty:      true,
			wantIsFull:       false,
			wantHasSpot:      false,
			wantLen:          0,
			wantErr:          false,
		},

		{
			name:             "Try dequeueing 3 times from a full queue",
			queue:            PlayersQueue{},
			noOfMethodsToRun: 3,
			queueLen:         2,
			wantIsDone:       true,
			wantIsEmpty:      true,
			wantIsFull:       false,
			wantHasSpot:      false,
			wantLen:          0,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.queue.NewQueue()

			for i := range tt.queueLen {
				q.Enqueue(components.OnlinePlayerStruct{
					Name: fmt.Sprintf("new game: %v", i),
				})
			}

			for range tt.noOfMethodsToRun {
				length := len(q.players)

				_, err := q.Dequeue()

				if len(q.players) == length-1 && err != nil {
					t.Errorf("Dequeue() length = %v, want length %v", len(q.players), length-1)
				}

				if (err != nil) != tt.wantErr && length == 0 {
					t.Errorf("Dequeue() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			if q.IsDone() != tt.wantIsDone {
				t.Errorf("IsDone() = %v, want %v", q.IsDone(), tt.wantIsDone)
			}

			if q.isEmpty() != tt.wantIsEmpty {
				t.Errorf("IsEmpty = %v, want %v", q.isEmpty(), tt.wantIsEmpty)
			}

			if q.isFull() != tt.wantIsFull {
				t.Errorf("isFull = %v, want %v", q.isFull(), tt.wantIsFull)
			}

			if q.HasSpot() != tt.wantHasSpot {
				t.Errorf("HasSpot = %v, want %v", q.HasSpot(), tt.wantHasSpot)
			}

			if len(q.players) != tt.wantLen {
				t.Errorf("Length of players in queue = %v, want %v", len(q.players), tt.wantLen)
			}
		})
	}
}
