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

func TestService_Complete(t *testing.T) {
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

	t.Run("completes task and adds points to user score", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{UserID: user.ID, TeamID: team.ID})
		task := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 10})

		result, err := b.svc.Complete(ctx, task.ID, user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.DoneByUserID == nil || *result.DoneByUserID != user.ID {
			t.Errorf("expected DoneByUserID %d, got %v", user.ID, result.DoneByUserID)
		}
		if result.DoneAt == nil {
			t.Error("expected DoneAt to be set")
		}

		var score int
		if err := b.tx.WithContext(ctx).Raw("SELECT score FROM users WHERE id = ?", user.ID).Scan(&score).Error; err != nil {
			t.Fatalf("reload user score: %v", err)
		}
		if score != 10 {
			t.Errorf("expected score 10, got %d", score)
		}
	})

	t.Run("returns ErrAlreadyCompleted if task already done", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{UserID: user.ID, TeamID: team.ID})
		now := time.Now()
		task := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{
			TeamID:       team.ID,
			Points:       5,
			DoneByUserID: &user.ID,
			DoneAt:       &now,
		})

		_, err := b.svc.Complete(ctx, task.ID, user.ID)
		if !errors.Is(err, tasks.ErrAlreadyCompleted) {
			t.Errorf("expected ErrAlreadyCompleted, got %v", err)
		}
	})

	t.Run("returns ErrNotMember if user not on team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		task := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 10})

		_, err := b.svc.Complete(ctx, task.ID, user.ID)
		if !errors.Is(err, tasks.ErrNotMember) {
			t.Errorf("expected ErrNotMember, got %v", err)
		}
	})

	t.Run("returns ErrTaskNotFound for nonexistent task", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		_, err := b.svc.Complete(ctx, 999999, user.ID)
		if !errors.Is(err, tasks.ErrTaskNotFound) {
			t.Errorf("expected ErrTaskNotFound, got %v", err)
		}
	})

	t.Run("accumulates score across multiple tasks", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{UserID: user.ID, TeamID: team.ID})
		task1 := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 10})
		task2 := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 20})

		if _, err := b.svc.Complete(ctx, task1.ID, user.ID); err != nil {
			t.Fatalf("Complete task1: %v", err)
		}
		if _, err := b.svc.Complete(ctx, task2.ID, user.ID); err != nil {
			t.Fatalf("Complete task2: %v", err)
		}

		var score int
		if err := b.tx.WithContext(ctx).Raw("SELECT score FROM users WHERE id = ?", user.ID).Scan(&score).Error; err != nil {
			t.Fatalf("reload user score: %v", err)
		}
		if score != 30 {
			t.Errorf("expected score 30, got %d", score)
		}
	})
}
