
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

-- name: GetUser :one
SELECT * FROM users
WHERE ID = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
  full_name
) VALUES (
  ?
)
RETURNING *;
