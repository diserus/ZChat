package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
	user.ID = uuid.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, email, name, encrypted_password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, user.ID, user.Email, user.Name, user.EncryptedPassword)
	return err
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, name, updated_at, created_at, encrypted_password
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.UpdatedAt, &u.CreatedAt, &u.EncryptedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, name, updated_at, created_at, encrypted_password
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.UpdatedAt, &u.CreatedAt, &u.EncryptedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

type PostgresTokenRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTokenRepository(db *pgxpool.Pool) *PostgresTokenRepository {
	return &PostgresTokenRepository{db: db}
}

func (r *PostgresTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, FALSE, NOW())
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt)
	return err
}

func (r *PostgresTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	t := RefreshToken{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get token by hash: %w", err)
	}
	return &t, nil
}

func (r *PostgresTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1`, id)
	return err
}
