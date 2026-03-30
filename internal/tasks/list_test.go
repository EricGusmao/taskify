package tasks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EricGusmao/taskify/internal/tasks"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func boolPtr(b bool) *bool { return &b }

func TestService_ListByTeam(t *testing.T) {
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

	t.Run("returns all tasks for team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		now := time.Now()

		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{
			TeamID:       team.ID,
			DoneByUserID: &user.ID,
			DoneAt:       &now,
		})

		page, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team.ID,
			Done:     nil,
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 3 {
			t.Errorf("expected total 3, got %d", page.Total)
		}
		if len(page.Data) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(page.Data))
		}
	})

	t.Run("filters done=true", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		now := time.Now()

		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{
			TeamID:       team.ID,
			DoneByUserID: &user.ID,
			DoneAt:       &now,
		})

		page, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team.ID,
			Done:     boolPtr(true),
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 1 {
			t.Errorf("expected total 1, got %d", page.Total)
		}
		if len(page.Data) != 1 {
			t.Errorf("expected 1 task, got %d", len(page.Data))
		}
		if page.Data[0].DoneAt == nil {
			t.Error("expected DoneAt to be set")
		}
	})

	t.Run("filters done=false", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		now := time.Now()

		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{
			TeamID:       team.ID,
			DoneByUserID: &user.ID,
			DoneAt:       &now,
		})

		page, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team.ID,
			Done:     boolPtr(false),
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 2 {
			t.Errorf("expected total 2, got %d", page.Total)
		}
		if len(page.Data) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(page.Data))
		}
		for _, task := range page.Data {
			if task.DoneAt != nil {
				t.Errorf("expected DoneAt to be nil, got %v", task.DoneAt)
			}
		}
	})

	t.Run("returns empty slice for team with no tasks", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		page, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team.ID,
			Done:     nil,
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 0 {
			t.Errorf("expected total 0, got %d", page.Total)
		}
		if len(page.Data) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(page.Data))
		}
	})

	t.Run("returns ErrTeamNotFound for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		_, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   999999,
			Done:     nil,
			Page:     1,
			PageSize: 20,
		})
		if !errors.Is(err, tasks.ErrTeamNotFound) {
			t.Errorf("expected ErrTeamNotFound, got %v", err)
		}
	})

	t.Run("paginates correctly", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		for i := 0; i < 5; i++ {
			dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		}

		page1, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team.ID,
			Done:     nil,
			Page:     1,
			PageSize: 2,
		})
		if err != nil {
			t.Fatalf("page 1 unexpected error: %v", err)
		}
		if len(page1.Data) != 2 {
			t.Errorf("page 1: expected 2 tasks, got %d", len(page1.Data))
		}
		if page1.Total != 5 {
			t.Errorf("page 1: expected total 5, got %d", page1.Total)
		}

		page3, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team.ID,
			Done:     nil,
			Page:     3,
			PageSize: 2,
		})
		if err != nil {
			t.Fatalf("page 3 unexpected error: %v", err)
		}
		if len(page3.Data) != 1 {
			t.Errorf("page 3: expected 1 task, got %d", len(page3.Data))
		}
		if page3.Total != 5 {
			t.Errorf("page 3: expected total 5, got %d", page3.Total)
		}
	})

	t.Run("does not leak tasks from other teams", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team1 := dbfactory.Team(ctx, t, b.tx, nil)
		team2 := dbfactory.Team(ctx, t, b.tx, nil)

		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team1.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team1.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team2.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team2.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team2.ID})

		page, err := b.svc.ListByTeam(ctx, tasks.ListByTeamInput{
			TeamID:   team1.ID,
			Done:     nil,
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 2 {
			t.Errorf("expected total 2, got %d", page.Total)
		}
		if len(page.Data) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(page.Data))
		}
		for _, task := range page.Data {
			if task.TeamID != team1.ID {
				t.Errorf("expected TeamID %d, got %d", team1.ID, task.TeamID)
			}
		}
	})
}
