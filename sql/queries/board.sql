-- name: CreateMove :exec
INSERT INTO board(board, move, white_time, black_time, match_id, created_at)
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

-- name: GetBoardForMove :one
SELECT board, white_time, black_time FROM board WHERE match_id = $1 AND move = $2;

-- name: GetAllMovesForMatch :many
SELECT move FROM board WHERE match_id = $1;