package app

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/auth/jwt"
	"ride-hail/internal/auth/repo"
	"ride-hail/internal/shared/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo *repo.AuthRepo
}

func NewAuthService(r *repo.AuthRepo) *AuthService {
	return &AuthService{repo: r}
}

func (s *AuthService) Register(ctx context.Context, email, password, role, name, phone string) (*models.User, error) {
	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	id := uuid.NewString()

	user := &models.User{
		ID:           id,
		Email:        email,
		Role:         role,
		Status:       "ACTIVE",
		PasswordHash: string(hash),
		Attrs: map[string]interface{}{
			"name":  name,
			"phone": phone,
		},
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, errors.New("user not registered")
		}
		return "", nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", nil, errors.New("invalid password")
	}

	exists, err := s.repo.CheckActiveToken(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}

	if exists {
		return "", nil, errors.New("user already logged in")
	}

	token, err := jwt.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, err
	}

	if err := s.repo.SaveActiveToken(ctx, user.ID, token); err != nil {
		return "", nil, err
	}

	return token, user, nil
}
