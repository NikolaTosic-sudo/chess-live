-- name: CreateMove :exec
INSERT INTO board(board, move, whiteTime, blackTime, match_id, created_at)
VALUES(
  $1,
  $1,
  $1,
  $1,
  $1,
  NOW()
);