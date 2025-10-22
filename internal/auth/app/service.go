package app

import (
	"context"
	"errors"
	"fmt"
	"ride-hail/internal/auth/jwt"
	"ride-hail/internal/auth/repo"
	"ride-hail/internal/shared/models"

	"github.com/google/uuid"
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
	if err != nil {
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
		return "", nil, errors.New("invalid credentials")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := jwt.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}
