-- name: CreateMatch :one
INSERT INTO matches(white, black, full_time, user_id, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  NOW()
) RETURNING id;

-- name: GetAllMatchesForUser :many
SELECT * FROM matches WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetMatchById :one
SELECT * FROM matches WHERE id = $1;