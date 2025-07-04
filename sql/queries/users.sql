-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserFromRefreshToken :one
SELECT * FROM users JOIN refresh_tokens ON users.id = refresh_tokens.user_id WHERE refresh_tokens.token = $1;

-- name: UpdateUserWithID :one
UPDATE users SET updated_at = NOW(), hashed_password = $2, email = $3 WHERE id = $1 RETURNING *;

-- name: PromoteToRedUserWithID :one
UPDATE users SET is_chirpy_red = true WHERE id = $1 RETURNING *;
