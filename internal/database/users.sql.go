// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: users.sql

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red
`

type CreateUserParams struct {
	Email          string
	HashedPassword string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser, arg.Email, arg.HashedPassword)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
		&i.IsChirpyRed,
	)
	return i, err
}

const deleteAllUsers = `-- name: DeleteAllUsers :exec
DELETE FROM users
`

func (q *Queries) DeleteAllUsers(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, deleteAllUsers)
	return err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password, is_chirpy_red FROM users WHERE email = $1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
		&i.IsChirpyRed,
	)
	return i, err
}

const getUserFromRefreshToken = `-- name: GetUserFromRefreshToken :one
SELECT id, users.created_at, users.updated_at, email, hashed_password, is_chirpy_red, token, refresh_tokens.created_at, refresh_tokens.updated_at, user_id, expires_at, revoked_at FROM users JOIN refresh_tokens ON users.id = refresh_tokens.user_id WHERE refresh_tokens.token = $1
`

type GetUserFromRefreshTokenRow struct {
	ID             uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Email          string
	HashedPassword string
	IsChirpyRed    bool
	Token          string
	CreatedAt_2    time.Time
	UpdatedAt_2    time.Time
	UserID         uuid.UUID
	ExpiresAt      time.Time
	RevokedAt      sql.NullTime
}

func (q *Queries) GetUserFromRefreshToken(ctx context.Context, token string) (GetUserFromRefreshTokenRow, error) {
	row := q.db.QueryRowContext(ctx, getUserFromRefreshToken, token)
	var i GetUserFromRefreshTokenRow
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
		&i.IsChirpyRed,
		&i.Token,
		&i.CreatedAt_2,
		&i.UpdatedAt_2,
		&i.UserID,
		&i.ExpiresAt,
		&i.RevokedAt,
	)
	return i, err
}

const promoteToRedUserWithID = `-- name: PromoteToRedUserWithID :one
UPDATE users SET is_chirpy_red = true WHERE id = $1 RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red
`

func (q *Queries) PromoteToRedUserWithID(ctx context.Context, id uuid.UUID) (User, error) {
	row := q.db.QueryRowContext(ctx, promoteToRedUserWithID, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
		&i.IsChirpyRed,
	)
	return i, err
}

const updateUserWithID = `-- name: UpdateUserWithID :one
UPDATE users SET updated_at = NOW(), hashed_password = $2, email = $3 WHERE id = $1 RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red
`

type UpdateUserWithIDParams struct {
	ID             uuid.UUID
	HashedPassword string
	Email          string
}

func (q *Queries) UpdateUserWithID(ctx context.Context, arg UpdateUserWithIDParams) (User, error) {
	row := q.db.QueryRowContext(ctx, updateUserWithID, arg.ID, arg.HashedPassword, arg.Email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Email,
		&i.HashedPassword,
		&i.IsChirpyRed,
	)
	return i, err
}
