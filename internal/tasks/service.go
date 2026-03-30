package tasks

import (
	"context"
	"fmt"
	"time"

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

// ListByTeamInput holds the input for the ListByTeam method.
type ListByTeamInput struct {
	TeamID   uint
	Done     *bool
	Page     int
	PageSize int
}

// TasksPage is the paginated result returned by ListByTeam.
type TasksPage struct {
	Data     []Task
	Page     int
	PageSize int
	Total    int64
}

// ListByTeam returns a paginated list of tasks for the given team.
func (s *Service) ListByTeam(ctx context.Context, input ListByTeamInput) (*TasksPage, error) {
	exists, err := s.repo.TeamExists(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("tasks.service.ListByTeam: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("tasks.service.ListByTeam: %w", ErrTeamNotFound)
	}

	list, total, err := s.repo.ListByTeam(ctx, input.TeamID, input.Done, input.Page, input.PageSize)
	if err != nil {
		return nil, fmt.Errorf("tasks.service.ListByTeam: %w", err)
	}

	return &TasksPage{
		Data:     list,
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    total,
	}, nil
}

// Complete marks the task as completed by the given user and adds the task's points to their score.
func (s *Service) Complete(ctx context.Context, taskID uint, userID uint) (*Task, error) {
	task, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("tasks.service.Complete: %w", err)
	}

	isMember, err := s.repo.IsMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, fmt.Errorf("tasks.service.Complete: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("tasks.service.Complete: %w", ErrNotMember)
	}

	if err := s.repo.Complete(ctx, taskID, userID, task.Points); err != nil {
		return nil, fmt.Errorf("tasks.service.Complete: %w", err)
	}

	s.logger.Info("task completed",
		zap.Uint("task_id", taskID),
		zap.Uint("user_id", userID),
		zap.Int("points", task.Points),
	)

	now := time.Now()
	task.DoneByUserID = &userID
	task.DoneAt = &now
	return task, nil
}
