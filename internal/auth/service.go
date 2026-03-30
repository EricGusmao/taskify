package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository is the persistence interface required by Service.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
}

// Service handles authentication business logic.
type Service struct {
	repo      UserRepository
	logger    *zap.Logger
	jwtSecret []byte
}

// NewService returns a Service with the provided repository, logger, and JWT signing secret.
func NewService(repo UserRepository, logger *zap.Logger, jwtSecret []byte) *Service {
	return &Service{repo: repo, logger: logger, jwtSecret: jwtSecret}
}

// RegisterInput holds the input for the Register method.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Register creates a new user with a hashed password.
// Returns ErrEmailTaken if the email is already registered.
func (s *Service) Register(ctx context.Context, input RegisterInput) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("auth.service.Register: hash password: %w", err)
	}

	user := &User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hash),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("auth.service.Register: %w", err)
	}

	s.logger.Debug("user registered", zap.String("email", input.Email))
	return user, nil
}

// LoginInput holds the credentials for the Login method.
type LoginInput struct {
	Email    string
	Password string
}

// Login validates credentials and returns a signed JWT on success.
// Returns ErrInvalidCredentials for unknown email or wrong password.
func (s *Service) Login(ctx context.Context, input LoginInput) (string, error) {
	user, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		return "", fmt.Errorf("auth.service.Login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(user.ID), 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("auth.service.Login: sign token: %w", err)
	}

	s.logger.Debug("user logged in", zap.String("email", input.Email))
	return token, nil
}
