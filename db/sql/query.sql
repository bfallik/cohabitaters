-- name: GetAuthor :one
SELECT * FROM authors
WHERE id = ? LIMIT 1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (
  name, bio
) VALUES (
  ?, ?
)
RETURNING *;

-- name: DeleteAuthor :exec
DELETE FROM authors
WHERE id = ?;

-- name: GetOauth2Token :one
SELECT * FROM oauth2_tokens
WHERE id = ? LIMIT 1;

-- name: CreateOauth2Token :one
INSERT INTO oauth2_tokens (
  id, token
) VALUES (
  ?, ?
)
RETURNING *;
