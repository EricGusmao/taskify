package tasks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EricGusmao/taskify/internal/tasks"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestService_Create(t *testing.T) {
	type testBundle struct {
		svc *tasks.Service
		tx  *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		ctx := context.Background()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		repo := tasks.NewRepository(tx)
		return &testBundle{
			svc: tasks.NewService(repo, zap.NewNop()),
			tx:  tx,
		}, ctx
	}

	t.Run("creates task for existing team", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		team := dbfactory.Team(ctx, t, bundle.tx, nil)

		task, err := bundle.svc.Create(ctx, tasks.CreateInput{
			TeamID: team.ID,
			Title:  "Fix login bug",
			Points: 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.ID == 0 {
			t.Error("expected task ID to be set")
		}
		if task.Title != "Fix login bug" {
			t.Errorf("expected title %q, got %q", "Fix login bug", task.Title)
		}
		if task.Points != 10 {
			t.Errorf("expected points 10, got %d", task.Points)
		}
		if task.TeamID != team.ID {
			t.Errorf("expected team_id %d, got %d", team.ID, task.TeamID)
		}
		if task.DoneByUserID != nil {
			t.Errorf("expected done_by_user_id to be nil, got %v", task.DoneByUserID)
		}
		if task.DoneAt != nil {
			t.Errorf("expected done_at to be nil, got %v", task.DoneAt)
		}
	})

	t.Run("returns ErrTeamNotFound for nonexistent team", func(t *testing.T) {
		t.Parallel()
		bundle, ctx := setup(t)

		_, err := bundle.svc.Create(ctx, tasks.CreateInput{
			TeamID: 999999,
			Title:  "Orphan task",
			Points: 5,
		})
		if !errors.Is(err, tasks.ErrTeamNotFound) {
			t.Errorf("expected ErrTeamNotFound, got %v", err)
		}
	})
}
