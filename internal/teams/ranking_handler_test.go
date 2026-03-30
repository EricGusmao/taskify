package teams_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EricGusmao/taskify/internal/teams"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestHandler_GetRanking(t *testing.T) {
	type testBundle struct {
		handler *teams.Handler
		e       *echo.Echo
		tx      *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		svc := teams.NewService(teams.NewRepository(tx), teams.NewUserRepository(tx), zap.NewNop())
		return &testBundle{
			handler: teams.NewHandler(svc),
			e:       newTestEcho(),
			tx:      tx,
		}, context.Background()
	}

	doRequest := func(t *testing.T, b *testBundle, teamIDStr string, query string) *httptest.ResponseRecorder {
		t.Helper()
		url := "/teams/" + teamIDStr + "/ranking"
		if query != "" {
			url += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := b.e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: teamIDStr}})
		if err := b.handler.GetRanking(c); err != nil {
			b.e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 200 with ranked members", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		u1 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 30})
		u2 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 10})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u1.ID})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u2.ID})

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), "")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp teams.RankingResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 2 {
			t.Errorf("expected total=2, got %d", resp.Total)
		}
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(resp.Data))
		}
		if resp.Data[0].Rank != 1 {
			t.Errorf("expected first rank=1, got %d", resp.Data[0].Rank)
		}
		if resp.Data[0].Score != 30 {
			t.Errorf("expected first score=30, got %d", resp.Data[0].Score)
		}
		if resp.Data[1].Rank != 2 {
			t.Errorf("expected second rank=2, got %d", resp.Data[1].Rank)
		}
	})

	t.Run("returns 200 with empty data array", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), "")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		body := rec.Body.String()
		var resp teams.RankingResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 0 {
			t.Errorf("expected total=0, got %d", resp.Total)
		}
		if resp.Data == nil {
			t.Error("expected non-nil empty slice, got null")
		}
		if !strings.Contains(body, `"data":[]`) {
			t.Errorf("expected data to be JSON array [], got: %s", body)
		}
	})

	t.Run("returns 404 for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, "999999", "")

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid team id", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, "abc", "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("uses default pagination", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), "")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp teams.RankingResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Page != 1 {
			t.Errorf("expected default page=1, got %d", resp.Page)
		}
		if resp.PageSize != 20 {
			t.Errorf("expected default page_size=20, got %d", resp.PageSize)
		}
	})
}
