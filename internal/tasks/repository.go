package tasks

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TaskRepository is the persistence interface required by Service.
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	TeamExists(ctx context.Context, teamID uint) (bool, error)
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
