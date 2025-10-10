-- name: CreateMatchUser :exec
INSERT INTO matches_users(match_id, user_id)
VALUES(
  $1,
  $2
);
