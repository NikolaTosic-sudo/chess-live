-- name: CreateMove :exec
INSERT INTO board(board, move, whiteTime, blackTime, match_id, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  $5,
  NOW()
);

-- name: GetNumberOfMovesPerMatch :one
SELECT COUNT(*) FROM board WHERE match_id = $1;