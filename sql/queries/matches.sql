-- name: CreateMatch :one
INSERT INTO matches(white, black, full_time, is_online, result, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  $5,
  NOW()
) RETURNING id;

-- name: GetAllMatchesForUser :many
SELECT * FROM matches WHERE id IN (
 SELECT match_id FROM matches_users WHERE user_id = $1
) ORDER BY created_at DESC LIMIT 30;

-- name: GetMatchById :one
SELECT * FROM matches WHERE id = $1;

-- name: UpdateMatchOnEnd :exec
UPDATE matches SET ended = true, result = $1
WHERE id = $2;
