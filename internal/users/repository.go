package users

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// UserRepository is the persistence interface required by Service.
type UserRepository interface {
	UpdateAvatarURL(ctx context.Context, userID uint, url string) error
	GetAvatarURL(ctx context.Context, userID uint) (string, error)
}

type gormUserRepository struct {
	db *gorm.DB
}

// NewRepository returns a UserRepository backed by GORM.
func NewRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// UpdateAvatarURL sets the avatar_url column for the given user.
func (r *gormUserRepository) UpdateAvatarURL(ctx context.Context, userID uint, url string) error {
	res := r.db.WithContext(ctx).
		Table("users").
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("avatar_url", url)
	if res.Error != nil {
		return fmt.Errorf("users.repository.UpdateAvatarURL: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("users.repository.UpdateAvatarURL: %w", ErrUserNotFound)
	}
	return nil
}

// GetAvatarURL returns the current avatar_url for the given user.
func (r *gormUserRepository) GetAvatarURL(ctx context.Context, userID uint) (string, error) {
	var url string
	err := r.db.WithContext(ctx).
		Table("users").
		Select("avatar_url").
		Where("id = ? AND deleted_at IS NULL", userID).
		Scan(&url).Error
	if err != nil {
		return "", fmt.Errorf("users.repository.GetAvatarURL: %w", err)
	}
	return url, nil
}
