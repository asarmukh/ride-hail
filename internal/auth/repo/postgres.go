package repo

import (
	"context"
	"errors"
	"ride-hail/internal/shared/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepo struct {
	db *pgxpool.Pool
}

func NewAuthRepo(db *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{db: db}
}

func (r *AuthRepo) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, role, status, password_hash, attrs)
		VALUES ($1, $2, $3, 'ACTIVE', $4, $5)
	`
	_, err := r.db.Exec(ctx, query, user.ID, user.Email, user.Role, user.PasswordHash, user.Attrs)
	return err
}

func (r *AuthRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, role, status, password_hash, attrs FROM users WHERE email=$1`
	row := r.db.QueryRow(ctx, query, email)

	user := &models.User{}
	err := row.Scan(&user.ID, &user.Email, &user.Role, &user.Status, &user.PasswordHash, &user.Attrs)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}
