package tasks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EricGusmao/taskify/internal/tasks"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"github.com/EricGusmao/taskify/internal/validate"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type echoValidator struct{}

func (echoValidator) Validate(i any) error { return validate.Struct(i) }

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = echoValidator{}
	return e
}

func TestHandler_Create(t *testing.T) {
	type testBundle struct {
		handler *tasks.Handler
		e       *echo.Echo
		tx      *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		svc := tasks.NewService(tasks.NewRepository(tx), zap.NewNop())
		return &testBundle{
			handler: tasks.NewHandler(svc),
			e:       newTestEcho(),
			tx:      tx,
		}, context.Background()
	}

	doRequest := func(t *testing.T, b *testBundle, teamID string, body any) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID+"/tasks", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := b.e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: teamID}})
		if err := b.handler.Create(c); err != nil {
			b.e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 201 with task data", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), tasks.CreateRequest{
			Title:  "Build API",
			Points: 20,
		})

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp tasks.CreateResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.ID == 0 {
			t.Error("expected ID > 0")
		}
		if resp.Title != "Build API" {
			t.Errorf("expected title %q, got %q", "Build API", resp.Title)
		}
		if resp.Points != 20 {
			t.Errorf("expected points 20, got %d", resp.Points)
		}
		if resp.TeamID != team.ID {
			t.Errorf("expected team_id %d, got %d", team.ID, resp.TeamID)
		}
	})

	t.Run("returns 404 for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, "999999", tasks.CreateRequest{
			Title:  "Orphan task",
			Points: 5,
		})

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on missing title", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), map[string]any{
			"points": 10,
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on zero points", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), map[string]any{
			"title":  "Task",
			"points": 0,
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on negative points", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), map[string]any{
			"title":  "Task",
			"points": -5,
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 400 on invalid team id", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, "abc", tasks.CreateRequest{
			Title:  "Task",
			Points: 10,
		})

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on title too long", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), tasks.CreateRequest{
			Title:  strings.Repeat("a", 256),
			Points: 10,
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on missing points", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), map[string]any{
			"title": "Task without points",
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})
}
