package tasks_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/EricGusmao/taskify/internal/middleware"
	"github.com/EricGusmao/taskify/internal/tasks"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestHandler_Complete(t *testing.T) {
	type testBundle struct {
		handler *tasks.Handler
		tx      *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		svc := tasks.NewService(tasks.NewRepository(tx), zap.NewNop())
		return &testBundle{
			handler: tasks.NewHandler(svc),
			tx:      tx,
		}, context.Background()
	}

	doRequest := func(t *testing.T, b *testBundle, taskID string, userID uint) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/tasks/"+taskID+"/complete", nil)
		rec := httptest.NewRecorder()
		e := newTestEcho()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: taskID}})
		c.Set(middleware.ContextKeyUserID, strconv.FormatUint(uint64(userID), 10))
		if err := b.handler.Complete(c); err != nil {
			e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 200 with completed task", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{UserID: user.ID, TeamID: team.ID})
		task := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 10})

		rec := doRequest(t, b, fmt.Sprintf("%d", task.ID), user.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 404 for nonexistent task", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		rec := doRequest(t, b, "999999", user.ID)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("returns 409 when task already completed", func(t *testing.T) {
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

		rec := doRequest(t, b, fmt.Sprintf("%d", task.ID), user.ID)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("returns 403 when user not a member", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		task := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 10})

		rec := doRequest(t, b, fmt.Sprintf("%d", task.ID), user.ID)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid task id", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		rec := doRequest(t, b, "abc", user.ID)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("returns 401 when jwt context key is absent", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		task := dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID, Points: 10})

		// Build request without setting the JWT context key.
		req := httptest.NewRequest(http.MethodPatch, "/tasks/"+fmt.Sprintf("%d", task.ID)+"/complete", nil)
		rec := httptest.NewRecorder()
		e := newTestEcho()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: fmt.Sprintf("%d", task.ID)}})
		// Deliberately do NOT set middleware.ContextKeyUserID.
		if err := b.handler.Complete(c); err != nil {
			e.HTTPErrorHandler(c, err)
		}

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}
