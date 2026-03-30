package tasks

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Service handles business logic for tasks.
type Service struct {
	repo   TaskRepository
	logger *zap.Logger
}

// NewService returns a Service with the provided repository and logger.
func NewService(repo TaskRepository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// CreateInput holds the input for the Create method.
type CreateInput struct {
	TeamID uint
	Title  string
	Points int
}

// Create creates a new task for the given team.
func (s *Service) Create(ctx context.Context, input CreateInput) (*Task, error) {
	exists, err := s.repo.TeamExists(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("tasks.service.Create: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("tasks.service.Create: %w", ErrTeamNotFound)
	}

	task := &Task{
		TeamID: input.TeamID,
		Title:  input.Title,
		Points: input.Points,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("tasks.service.Create: %w", err)
	}

	s.logger.Info("task created", zap.Uint("team_id", input.TeamID), zap.String("title", input.Title))
	return task, nil
}
