-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(
  token,
  created_at,
  updated_at,
  user_id,
  expires_at
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5
) RETURNING *;

-- name: RefreshToken :one
UPDATE refresh_tokens
SET token = $1, expires_at = $2, updated_at = $3
WHERE user_id = $4
RETURNING *;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET revoked_at = $1, updated_at = $1
WHERE token = $2;

-- name: SearchForToken :one
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: DeleteTokens :exec
DELETE FROM refresh_tokens;