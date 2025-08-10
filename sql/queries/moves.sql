-- name: CreateMove :exec
INSERT INTO moves(board, move, white_time, black_time, match_id, created_at)
VALUES(
  $1,
  $2,
  $3,
  $4,
  $5,
  NOW()
);

-- name: GetNumberOfMovesPerMatch :one
SELECT COUNT(*) FROM moves WHERE match_id = $1;

-- name: GetBoardForMove :one
SELECT board, white_time, black_time FROM moves WHERE match_id = $1 AND move = $2;

-- name: GetAllMovesForMatch :many
SELECT move FROM moves WHERE match_id = $1;

-- name: UpdateBoardForMove :exec
UPDATE moves SET board = $1 WHERE match_id = $2 AND move = $3;

-- name: GetLatestMoveForMatch :one
SELECT move, match_id FROM moves WHERE match_id = $1
ORDER BY created_at DESC;