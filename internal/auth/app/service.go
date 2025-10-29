package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/auth/jwt"
	"ride-hail/internal/auth/repo"
	"ride-hail/internal/shared/models"
	"ride-hail/internal/shared/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo   *repo.AuthRepo
	logger *util.Logger
}

func NewAuthService(r *repo.AuthRepo, logger *util.Logger) *AuthService {
	return &AuthService{repo: r, logger: logger}
}

func (s *AuthService) Register(ctx context.Context, email, password, role, name, phone string) (*models.User, error) {
	instance := "AuthService.Register"
	start := time.Now()

	s.logger.Info(instance, fmt.Sprintf("attempting to register new user [email=%s, role=%s]", email, role))

	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error(instance, fmt.Errorf("failed to check existing user: %w", err))
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		s.logger.Warn(instance, fmt.Sprintf("user with email %s already exists", email))
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error(instance, fmt.Errorf("failed to hash password: %w", err))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

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

	s.logger.Info(instance, fmt.Sprintf("creating user record in DB [id=%s]", id))
	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logger.Error(instance, fmt.Errorf("failed to create user in DB: %w", err))
		return nil, err
	}

	s.logger.OK(instance, fmt.Sprintf("user registered successfully [user_id=%s, email=%s]", id, email))
	s.logger.Info(instance, fmt.Sprintf("registration completed in %dms", time.Since(start).Milliseconds()))

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	instance := "AuthService.Login"
	start := time.Now()

	s.logger.Info(instance, fmt.Sprintf("user attempting login [email=%s]", email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn(instance, fmt.Sprintf("login failed: user not registered [email=%s]", email))
			return "", nil, errors.New("user not registered")
		}
		s.logger.Error(instance, fmt.Errorf("failed to query user: %w", err))
		return "", nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.logger.Warn(instance, fmt.Sprintf("invalid password for user [email=%s]", email))
		return "", nil, errors.New("invalid password")
	}

	exists, err := s.repo.CheckActiveToken(ctx, user.ID)
	if err != nil {
		s.logger.Error(instance, fmt.Errorf("failed to check active token: %w", err))
		return "", nil, err
	}

	if exists {
		s.logger.Warn(instance, fmt.Sprintf("user already logged in [user_id=%s]", user.ID))
		return "", nil, errors.New("user already logged in")
	}

	token, err := jwt.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		s.logger.Error(instance, fmt.Errorf("failed to generate token: %w", err))
		return "", nil, err
	}

	if err := s.repo.SaveActiveToken(ctx, user.ID, token); err != nil {
		s.logger.Error(instance, fmt.Errorf("failed to save active token: %w", err))
		return "", nil, err
	}

	s.logger.OK(instance, fmt.Sprintf("user login successful [user_id=%s, role=%s]", user.ID, user.Role))
	s.logger.Info(instance, fmt.Sprintf("login completed in %dms", time.Since(start).Milliseconds()))

	return token, user, nil
}
