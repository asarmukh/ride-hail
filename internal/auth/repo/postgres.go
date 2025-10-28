package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ride-hail/internal/shared/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepo struct {
	db *pgxpool.Pool
}

func NewAuthRepo(db *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{db: db}
}

func (r *AuthRepo) CreateUser(ctx context.Context, user *models.User) error {
	attrsJSON, err := json.Marshal(user.Attrs)
	if err != nil {
		return fmt.Errorf("failed to marshall attrs: %w", err)
	}
	query := `
		INSERT INTO users (id, email, role, status, password_hash, attrs)
		VALUES ($1, $2, $3, 'ACTIVE', $4, $5)
	`
	if _, err := r.db.Exec(ctx, query, user.ID, user.Email, user.Role, user.PasswordHash, attrsJSON); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *AuthRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, role, status, password_hash, attrs FROM users WHERE email=$1`
	row := r.db.QueryRow(ctx, query, email)

	user := &models.User{}
	var attrs []byte

	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.Status, &user.PasswordHash, &attrs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	if len(attrs) > 0 {
		if err := json.Unmarshal(attrs, &user.Attrs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user attrs: %w", err)
		}
	}
	return user, nil
}

func (r *AuthRepo) CheckActiveToken(ctx context.Context, userID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM active_tokens WHERE user_id=$1`, userID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *AuthRepo) SaveActiveToken(ctx context.Context, userID, token string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO active_tokens (user_id, token)
		VALUES ($1, $2)
		ON CONFLICT (user_id)
		DO UPDATE SET token = EXCLUDED.token, created_at = NOW()
	`, userID, token)
	return err
}
