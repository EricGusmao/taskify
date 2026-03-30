package tasks_test

import (
	"context"
	"encoding/json"
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

func TestHandler_ListByTeam(t *testing.T) {
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

	doRequest := func(t *testing.T, b *testBundle, teamID string, query string, userID uint) *httptest.ResponseRecorder {
		t.Helper()
		url := "/teams/" + teamID + "/tasks"
		if query != "" {
			url += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		e := newTestEcho()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: teamID}})
		c.Set(middleware.ContextKeyUserID, strconv.FormatUint(uint64(userID), 10))
		if err := b.handler.ListByTeam(c); err != nil {
			e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 200 with tasks", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})

		rec := doRequest(t, b, strconv.FormatUint(uint64(team.ID), 10), "", user.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		type listResp struct {
			Data     []map[string]any `json:"data"`
			Total    int64            `json:"total"`
			Page     int              `json:"page"`
			PageSize int              `json:"page_size"`
		}
		var resp listResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 2 {
			t.Errorf("expected total 2, got %d", resp.Total)
		}
	})

	t.Run("returns 200 with done=true filter", func(t *testing.T) {
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

		rec := doRequest(t, b, strconv.FormatUint(uint64(team.ID), 10), "done=true", user.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		type listResp struct {
			Data     []map[string]any `json:"data"`
			Total    int64            `json:"total"`
			Page     int              `json:"page"`
			PageSize int              `json:"page_size"`
		}
		var resp listResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 1 {
			t.Errorf("expected total 1, got %d", resp.Total)
		}
	})

	t.Run("returns 200 with done=false filter", func(t *testing.T) {
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

		rec := doRequest(t, b, strconv.FormatUint(uint64(team.ID), 10), "done=false", user.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		type listResp struct {
			Data     []map[string]any `json:"data"`
			Total    int64            `json:"total"`
			Page     int              `json:"page"`
			PageSize int              `json:"page_size"`
		}
		var resp listResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 2 {
			t.Errorf("expected total 2, got %d", resp.Total)
		}
	})

	t.Run("returns 404 for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		rec := doRequest(t, b, "999999", "", user.ID)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid team id", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		rec := doRequest(t, b, "abc", "", user.ID)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("respects page_size param", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})
		dbfactory.Task(ctx, t, b.tx, &dbfactory.TaskOpts{TeamID: team.ID})

		rec := doRequest(t, b, strconv.FormatUint(uint64(team.ID), 10), "page_size=2", user.ID)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		type listResp struct {
			Data     []map[string]any `json:"data"`
			Total    int64            `json:"total"`
			Page     int              `json:"page"`
			PageSize int              `json:"page_size"`
		}
		var resp listResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(resp.Data) != 2 {
			t.Errorf("expected 2 items in data, got %d", len(resp.Data))
		}
		if resp.Total != 3 {
			t.Errorf("expected total 3, got %d", resp.Total)
		}
	})
}
