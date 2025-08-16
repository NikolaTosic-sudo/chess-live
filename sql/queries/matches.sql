-- name: CreateMatch :one
INSERT INTO matches(white, black, full_time, user_id, is_online, created_at, result)
VALUES(
  $1,
  $2,
  $3,
  $4,
  $5,
  NOW(),
  "0-0"
) RETURNING id;

-- name: GetAllMatchesForUser :many
SELECT * FROM matches WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetMatchById :one
SELECT * FROM matches WHERE id = $1;

-- name: UpdateMatchOnEnd :exec
UPDATE matches SET ended = true, result = $1
WHERE id = $2;