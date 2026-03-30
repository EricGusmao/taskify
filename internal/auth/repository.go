package auth

import (
	"context"
	"errors"
	"fmt"

	mysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository returns a UserRepository backed by GORM.
func NewUserRepository(db *gorm.DB) *gormUserRepository {
	return &gormUserRepository{db: db}
}

// Create inserts the user into the database.
// Returns ErrEmailTaken if the email is already registered.
func (r *gormUserRepository) Create(ctx context.Context, user *User) error {
	if err := gorm.G[User](r.db).Create(ctx, user); err != nil {
		if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
			return ErrEmailTaken
		}
		return fmt.Errorf("auth.repository.Create: %w", err)
	}
	return nil
}
