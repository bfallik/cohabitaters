
-- name: GetToken :one
SELECT token FROM users u
INNER JOIN sessions s
ON u.id = s.user_id
WHERE s.id = ? LIMIT 1;

-- name: UpdateTokenBySession :exec
UPDATE users
SET token = ?
WHERE (
  SELECT user_id
  FROM sessions
  WHERE sessions.id = ?
  AND users.id = user_id
);

-- name: GetSession :one
SELECT * FROM sessions
WHERE ID = ? LIMIT 1;

-- name: InsertSession :one
INSERT INTO sessions (
  id, user_id
) VALUES (
  ?, ?
)
RETURNING *;

-- name: UpsertSession :one
INSERT INTO sessions (
  id, user_id
) VALUES (
  ?, ?
)
ON CONFLICT(id, user_id) DO UPDATE SET
  id=excluded.id
RETURNING *;

-- name: ExpireSession :exec
UPDATE sessions
SET is_logged_in = false
WHERE id = ?;

-- name: UpdateGoogleForceApproval :exec
UPDATE sessions
SET google_force_approval = ?
WHERE id = ?;

-- name: UpdateContactGroupsJSON :exec
UPDATE sessions
SET contact_groups_json = ?
WHERE id = ?;

-- name: UpdateSelectedResourceName :exec
UPDATE sessions
SET selected_resource_name = ?
WHERE id = ?;

-- name: GetUser :one
SELECT * FROM users
WHERE id = ? LIMIT 1;

-- name: GetUserBySession :one
SELECT u.* FROM users u
INNER JOIN sessions s
WHERE u.id = s.user_id
AND s.id = ? LIMIT 1;

-- name: InsertUser :one
INSERT INTO users
(
  sub,
  name,
  picture
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: UpsertUser :one
INSERT INTO users
(
  sub,
  name,
  picture
) VALUES (
  ?, ?, ?
)
ON CONFLICT(sub) DO UPDATE SET
  name=excluded.name,
  picture=excluded.picture
RETURNING *;
