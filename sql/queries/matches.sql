-- name: CreateMatch :one
INSERT INTO matches(white, black, timeOption, userId, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  NOW()
) RETURNING *;