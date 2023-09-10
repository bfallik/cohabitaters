
-- name: GetToken :one
SELECT * FROM tokens
WHERE id = ? LIMIT 1;

-- name: CreateToken :one
INSERT INTO tokens (
  id, user_id, token
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions
WHERE ID = ? LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (
  id, user_id
) VALUES (
  ?, ?
)
RETURNING *;

-- name: ExpireSession :exec
UPDATE sessions SET is_logged_in = false
WHERE id = ?;

-- name: GetUser :one
SELECT * FROM users
WHERE id = ? LIMIT 1;

-- name: GetUserBySub :one
SELECT * FROM users
WHERE sub = ? LIMIT 1;


-- name: CreateUser :one
INSERT OR REPLACE INTO users (
  full_name,
  sub
) VALUES (
  ?, ?
)
RETURNING *;
