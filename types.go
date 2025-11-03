package main

import (
	"github.com/NikolaTosic-sudo/chess-live/internal/database"
	"github.com/NikolaTosic-sudo/chess-live/internal/matches"
	"github.com/google/uuid"
)

type appConfig struct {
	database *database.Queries
	secret   string
	users    map[uuid.UUID]User
	Matches  matches.Matches
}

type User struct {
	Id    uuid.UUID
	Name  string
	Email string
}
