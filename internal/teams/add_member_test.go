package teams_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EricGusmao/taskify/internal/teams"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestService_AddMember(t *testing.T) {
	type testBundle struct {
		svc *teams.Service
		tx  *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		ctx := context.Background()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		repo := teams.NewRepository(tx)
		userRepo := teams.NewUserRepository(tx)
		return &testBundle{
			svc: teams.NewService(repo, userRepo, zap.NewNop()),
			tx:  tx,
		}, ctx
	}

	t.Run("adds member successfully", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		user := dbfactory.User(ctx, t, bundle.tx, nil)
		team := dbfactory.Team(ctx, t, bundle.tx, nil)

		member, err := bundle.svc.AddMember(ctx, teams.AddMemberInput{
			TeamID: team.ID,
			UserID: user.ID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if member.TeamID != team.ID {
			t.Errorf("expected TeamID %d, got %d", team.ID, member.TeamID)
		}
		if member.UserID != user.ID {
			t.Errorf("expected UserID %d, got %d", user.ID, member.UserID)
		}
	})

	t.Run("returns ErrTeamNotFound for nonexistent team", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		user := dbfactory.User(ctx, t, bundle.tx, nil)

		_, err := bundle.svc.AddMember(ctx, teams.AddMemberInput{
			TeamID: 999999,
			UserID: user.ID,
		})
		if !errors.Is(err, teams.ErrTeamNotFound) {
			t.Errorf("expected ErrTeamNotFound, got %v", err)
		}
	})

	t.Run("returns ErrUserNotFound for nonexistent user", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		team := dbfactory.Team(ctx, t, bundle.tx, nil)

		_, err := bundle.svc.AddMember(ctx, teams.AddMemberInput{
			TeamID: team.ID,
			UserID: 999999,
		})
		if !errors.Is(err, teams.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("returns ErrAlreadyMember on duplicate", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		user := dbfactory.User(ctx, t, bundle.tx, nil)
		team := dbfactory.Team(ctx, t, bundle.tx, nil)

		_, err := bundle.svc.AddMember(ctx, teams.AddMemberInput{
			TeamID: team.ID,
			UserID: user.ID,
		})
		if err != nil {
			t.Fatalf("unexpected error on first add: %v", err)
		}

		_, err = bundle.svc.AddMember(ctx, teams.AddMemberInput{
			TeamID: team.ID,
			UserID: user.ID,
		})
		if !errors.Is(err, teams.ErrAlreadyMember) {
			t.Errorf("expected ErrAlreadyMember, got %v", err)
		}
	})
}
