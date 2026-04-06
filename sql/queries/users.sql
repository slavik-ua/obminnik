-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash)
VALUES ($1, $2, $3);

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;