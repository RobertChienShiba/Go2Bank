-- name: CreateUser :one
INSERT INTO users (
  username,
  hashed_password,
  full_name,
  email
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET
  hashed_password = COALESCE(sqlc.narg(hashed_password), hashed_password),
  password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
  full_name = COALESCE(sqlc.narg(full_name), full_name),
  email = COALESCE(sqlc.narg(email), email)
WHERE
  username = sqlc.arg(username)
RETURNING *;

-- name: UpsertUser :one
INSERT INTO users (
  username,
  hashed_password,
  full_name,
  email,
  provider
) VALUES (
  $1, $2, $3, $4, $5
) ON CONFLICT (username)
DO UPDATE 
SET 
  email = EXCLUDED.email, 
  full_name = EXCLUDED.full_name,
  hashed_password = EXCLUDED.hashed_password
RETURNING *;