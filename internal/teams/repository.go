package teams

import (
	"context"
	"errors"
	"fmt"

	"github.com/EricGusmao/taskify/internal/auth"
	mysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Repository handles persistence for the teams slice.
type Repository interface {
	Create(ctx context.Context, team *Team) error
	FindByID(ctx context.Context, id uint) (*Team, error)
	AddMember(ctx context.Context, teamID, userID uint) error
	ListMembers(ctx context.Context, teamID uint, page, pageSize int) ([]MemberWithUser, int64, error)
	ListRanking(ctx context.Context, teamID uint, page, pageSize int) ([]MemberWithUser, int64, error)
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

// FindByID returns the team with the given ID.
func (r *gormRepository) FindByID(ctx context.Context, id uint) (*Team, error) {
	team, err := gorm.G[Team](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("teams.repository.FindByID: %w", ErrTeamNotFound)
		}
		return nil, fmt.Errorf("teams.repository.FindByID: %w", err)
	}
	return &team, nil
}

// AddMember inserts a row into the members join table directly to allow
// duplicate-key detection. GORM's Association.Append uses ON CONFLICT DO NOTHING,
// which would silently swallow duplicates.
func (r *gormRepository) AddMember(ctx context.Context, teamID, userID uint) error {
	member := &Member{TeamID: teamID, UserID: userID}
	if err := gorm.G[Member](r.db).Create(ctx, member); err != nil {
		if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
			return fmt.Errorf("teams.repository.AddMember: %w", ErrAlreadyMember)
		}
		return fmt.Errorf("teams.repository.AddMember: %w", err)
	}
	return nil
}

// ListMembers returns a paginated list of team members with their user details.
func (r *gormRepository) ListMembers(ctx context.Context, teamID uint, page, pageSize int) ([]MemberWithUser, int64, error) {
	var count int64
	countSQL := `
		SELECT COUNT(*)
		FROM members m
		JOIN users u ON u.id = m.user_id AND u.deleted_at IS NULL
		WHERE m.team_id = ?`
	if err := r.db.WithContext(ctx).Raw(countSQL, teamID).Scan(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("teams.repository.ListMembers: count: %w", err)
	}

	results := make([]MemberWithUser, 0)
	offset := (page - 1) * pageSize
	listSQL := `
		SELECT m.user_id, u.name, u.email, u.score
		FROM members m
		JOIN users u ON u.id = m.user_id AND u.deleted_at IS NULL
		WHERE m.team_id = ?
		ORDER BY u.score DESC
		LIMIT ? OFFSET ?`
	if err := r.db.WithContext(ctx).Raw(listSQL, teamID, pageSize, offset).Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("teams.repository.ListMembers: list: %w", err)
	}

	return results, count, nil
}

// ListRanking returns a paginated list of team members ordered by score descending.
// Ties are broken by user ID ascending for deterministic ordering.
func (r *gormRepository) ListRanking(ctx context.Context, teamID uint, page, pageSize int) ([]MemberWithUser, int64, error) {
	var count int64
	countSQL := `
		SELECT COUNT(*)
		FROM members m
		JOIN users u ON u.id = m.user_id AND u.deleted_at IS NULL
		WHERE m.team_id = ?`
	if err := r.db.WithContext(ctx).Raw(countSQL, teamID).Scan(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("teams.repository.ListRanking: count: %w", err)
	}

	results := make([]MemberWithUser, 0)
	offset := (page - 1) * pageSize
	listSQL := `
		SELECT m.user_id, u.name, u.email, u.score
		FROM members m
		JOIN users u ON u.id = m.user_id AND u.deleted_at IS NULL
		WHERE m.team_id = ?
		ORDER BY u.score DESC, u.id ASC
		LIMIT ? OFFSET ?`
	if err := r.db.WithContext(ctx).Raw(listSQL, teamID, pageSize, offset).Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("teams.repository.ListRanking: %w", err)
	}

	return results, count, nil
}

// gormUserRepository checks user existence.
type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository returns a UserRepository backed by GORM.
func NewUserRepository(db *gorm.DB) *gormUserRepository {
	return &gormUserRepository{db: db}
}

// Exists reports whether a user with the given ID exists.
func (r *gormUserRepository) Exists(ctx context.Context, id uint) (bool, error) {
	count, err := gorm.G[auth.User](r.db).Where("id = ?", id).Count(ctx, "*")
	if err != nil {
		return false, fmt.Errorf("teams.userRepository.Exists: %w", err)
	}
	return count > 0, nil
}
