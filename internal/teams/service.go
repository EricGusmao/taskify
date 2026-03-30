package teams

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// TeamRepository is the persistence interface required by Service.
type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
}

// Service handles business logic for teams.
type Service struct {
	repo   TeamRepository
	logger *zap.Logger
}

// NewService returns a Service with the provided repository and logger.
func NewService(repo TeamRepository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// CreateInput holds the input for the Create method.
type CreateInput struct {
	Name string
}

// Create creates a new team.
func (s *Service) Create(ctx context.Context, input CreateInput) (*Team, error) {
	team := &Team{Name: input.Name}
	if err := s.repo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("teams.service.Create: %w", err)
	}

	s.logger.Debug("team created", zap.String("name", input.Name))
	return team, nil
}
