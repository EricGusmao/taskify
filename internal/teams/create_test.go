package teams_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EricGusmao/taskify/internal/teams"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestService_Create(t *testing.T) {
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
		return &testBundle{
			svc: teams.NewService(repo, zap.NewNop()),
			tx:  tx,
		}, ctx
	}

	t.Run("creates team successfully", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		team, err := bundle.svc.Create(ctx, teams.CreateInput{Name: "Team Alpha"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if team.ID == 0 {
			t.Error("expected team ID to be set")
		}
		if team.Name != "Team Alpha" {
			t.Errorf("expected name Team Alpha, got %s", team.Name)
		}

		count, err := gorm.G[teams.Team](bundle.tx).Where("name = ?", "Team Alpha").Count(ctx, "*")
		if err != nil {
			t.Fatalf("count teams: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 team in DB, got %d", count)
		}
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		_, err := bundle.svc.Create(ctx, teams.CreateInput{Name: "Duplicate Team"})
		if err != nil {
			t.Fatalf("unexpected error on first create: %v", err)
		}

		_, err = bundle.svc.Create(ctx, teams.CreateInput{Name: "Duplicate Team"})
		if !errors.Is(err, teams.ErrNameTaken) {
			t.Errorf("expected ErrNameTaken, got %v", err)
		}
	})
}
