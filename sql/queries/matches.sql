-- name: CreateMatch :one
INSERT INTO matches(white, black, full_time, user_id, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  NOW()
) RETURNING id;