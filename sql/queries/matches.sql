-- name: CreateMatch :one
INSERT INTO matches(white, black, timer, userId, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  NOW(),
) RETURNING *;