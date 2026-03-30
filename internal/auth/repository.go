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

// FindByEmail looks up a user by email address.
// Returns ErrInvalidCredentials if no user with that email exists.
func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	u, err := gorm.G[User](r.db).Where("email = ?", email).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth.repository.FindByEmail: %w", err)
	}
	return &u, nil
}
