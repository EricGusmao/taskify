package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TaskRepository is the persistence interface required by Service.
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	TeamExists(ctx context.Context, teamID uint) (bool, error)
	FindByID(ctx context.Context, taskID uint) (*Task, error)
	IsMember(ctx context.Context, teamID uint, userID uint) (bool, error)
	Complete(ctx context.Context, taskID uint, userID uint, points int) error
	ListByTeam(ctx context.Context, teamID uint, done *bool, page, pageSize int) ([]Task, int64, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a TaskRepository backed by GORM.
func NewRepository(db *gorm.DB) TaskRepository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, task *Task) error {
	if err := gorm.G[Task](r.db).Create(ctx, task); err != nil {
		return fmt.Errorf("tasks.repository.Create: %w", err)
	}
	return nil
}

// TeamExists checks whether a team with the given ID exists without importing the teams package.
func (r *gormRepository) TeamExists(ctx context.Context, teamID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM teams WHERE id = ? AND deleted_at IS NULL", teamID).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("tasks.repository.TeamExists: %w", err)
	}
	return count > 0, nil
}

// FindByID fetches a task by its primary key.
func (r *gormRepository) FindByID(ctx context.Context, taskID uint) (*Task, error) {
	task, err := gorm.G[Task](r.db).Where("id = ?", taskID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tasks.repository.FindByID: %w", ErrTaskNotFound)
		}
		return nil, fmt.Errorf("tasks.repository.FindByID: %w", err)
	}
	return &task, nil
}

// IsMember checks whether a user is a member of the given team.
func (r *gormRepository) IsMember(ctx context.Context, teamID uint, userID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM members WHERE team_id = ? AND user_id = ?", teamID, userID).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("tasks.repository.IsMember: %w", err)
	}
	return count > 0, nil
}

// ListByTeam returns a paginated list of tasks for the given team, with an optional done filter.
func (r *gormRepository) ListByTeam(ctx context.Context, teamID uint, done *bool, page, pageSize int) ([]Task, int64, error) {
	base := r.db.WithContext(ctx).Model(&Task{}).Where("team_id = ? AND deleted_at IS NULL", teamID)
	if done != nil {
		if *done {
			base = base.Where("done_at IS NOT NULL")
		} else {
			base = base.Where("done_at IS NULL")
		}
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("tasks.repository.ListByTeam: %w", err)
	}

	var list []Task
	offset := (page - 1) * pageSize
	if err := base.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("tasks.repository.ListByTeam: %w", err)
	}

	return list, total, nil
}

// Complete atomically marks a task as done and increments the user's score.
func (r *gormRepository) Complete(ctx context.Context, taskID uint, userID uint, points int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&Task{}).
			Where("id = ? AND done_by_user_id IS NULL", taskID).
			Updates(map[string]any{"done_by_user_id": userID, "done_at": now})
		if result.Error != nil {
			return fmt.Errorf("tasks.repository.Complete: update task: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("tasks.repository.Complete: %w", ErrAlreadyCompleted)
		}

		if err := tx.Table("users").Where("id = ?", userID).
			Update("score", gorm.Expr("score + ?", points)).Error; err != nil {
			return fmt.Errorf("tasks.repository.Complete: update score: %w", err)
		}
		return nil
	})
}
