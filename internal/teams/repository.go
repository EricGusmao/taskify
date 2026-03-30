package teams

import (
	"context"
	"errors"
	"fmt"

	mysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Repository handles persistence for the teams slice.
type Repository interface {
	Create(ctx context.Context, team *Team) error
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a Repository backed by GORM.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, team *Team) error {
	if err := gorm.G[Team](r.db).Create(ctx, team); err != nil {
		if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
			return fmt.Errorf("teams.repository.Create: %w", ErrNameTaken)
		}
		return fmt.Errorf("teams.repository.Create: %w", err)
	}
	return nil
}
