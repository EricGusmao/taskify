package teams

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// TeamRepository is the persistence interface required by Service.
type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	FindByID(ctx context.Context, id uint) (*Team, error)
	AddMember(ctx context.Context, teamID, userID uint) error
	ListMembers(ctx context.Context, teamID uint, page, pageSize int) ([]MemberWithUser, int64, error)
}

// UserRepository checks user existence.
type UserRepository interface {
	Exists(ctx context.Context, id uint) (bool, error)
}

// Service handles business logic for teams.
type Service struct {
	repo     TeamRepository
	userRepo UserRepository
	logger   *zap.Logger
}

// NewService returns a Service with the provided repositories and logger.
func NewService(repo TeamRepository, userRepo UserRepository, logger *zap.Logger) *Service {
	return &Service{repo: repo, userRepo: userRepo, logger: logger}
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

// AddMemberInput holds the input for the AddMember method.
type AddMemberInput struct {
	TeamID uint
	UserID uint
}

// AddMember adds a user to a team.
func (s *Service) AddMember(ctx context.Context, input AddMemberInput) (*Member, error) {
	if _, err := s.repo.FindByID(ctx, input.TeamID); err != nil {
		return nil, fmt.Errorf("teams.service.AddMember: %w", err)
	}

	exists, err := s.userRepo.Exists(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("teams.service.AddMember: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("teams.service.AddMember: %w", ErrUserNotFound)
	}

	if err := s.repo.AddMember(ctx, input.TeamID, input.UserID); err != nil {
		return nil, fmt.Errorf("teams.service.AddMember: %w", err)
	}

	s.logger.Debug("member added", zap.Uint("team_id", input.TeamID), zap.Uint("user_id", input.UserID))
	return &Member{TeamID: input.TeamID, UserID: input.UserID}, nil
}

// ListMembersInput holds the input for the ListMembers method.
type ListMembersInput struct {
	TeamID   uint
	Page     int
	PageSize int
}

// MembersPage is the paginated result of ListMembers.
type MembersPage struct {
	Data     []MemberWithUser
	Page     int
	PageSize int
	Total    int64
}

// ListMembers returns a paginated list of members for a team.
func (s *Service) ListMembers(ctx context.Context, input ListMembersInput) (*MembersPage, error) {
	if _, err := s.repo.FindByID(ctx, input.TeamID); err != nil {
		return nil, fmt.Errorf("teams.service.ListMembers: %w", err)
	}

	members, total, err := s.repo.ListMembers(ctx, input.TeamID, input.Page, input.PageSize)
	if err != nil {
		return nil, fmt.Errorf("teams.service.ListMembers: %w", err)
	}

	return &MembersPage{
		Data:     members,
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    total,
	}, nil
}
