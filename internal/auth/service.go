package auth

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository is the persistence interface required by Service.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
}

// Service handles authentication business logic.
type Service struct {
	repo   UserRepository
	logger *zap.Logger
}

// NewService returns a Service with the provided repository and logger.
func NewService(repo UserRepository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
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
