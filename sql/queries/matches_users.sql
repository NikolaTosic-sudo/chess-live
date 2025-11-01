-- name: CreateMatchUser :exec
INSERT INTO matches_users(match_id, user_id)
VALUES(
  $1,
  $2
);

-- name: DeleteMatchUser :exec
DELETE FROM matches_users WHERE user_id = $1 AND match_id = $2;
