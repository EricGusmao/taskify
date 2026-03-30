package teams_test

import (
	"bytes"
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

	doRequest := func(t *testing.T, b *testBundle, body any) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/teams", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := b.e.NewContext(req, rec)
		if err := b.handler.Create(c); err != nil {
			b.e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 201 with team data", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, teams.CreateRequest{
			Name: "Team Bravo",
		})

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		var resp teams.CreateResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.ID == 0 {
			t.Error("expected ID > 0")
		}
		if resp.Name != "Team Bravo" {
			t.Errorf("expected name Team Bravo, got %s", resp.Name)
		}
	})

	t.Run("returns 409 on duplicate name", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		dbfactory.Team(ctx, t, b.tx, &dbfactory.TeamOpts{
			Name: "Duplicate",
		})

		rec := doRequest(t, b, teams.CreateRequest{
			Name: "Duplicate",
		})

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on missing name", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, struct{}{})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on too long name", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		name := strings.Repeat("a", 256)

		rec := doRequest(t, b, teams.CreateRequest{
			Name: name,
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})
}

func TestHandler_AddMember(t *testing.T) {
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

	doRequest := func(t *testing.T, b *testBundle, teamID string, body any) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID+"/members", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := b.e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: teamID}})
		if err := b.handler.AddMember(c); err != nil {
			b.e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 201 with member data", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), teams.AddMemberRequest{UserID: user.ID})

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp teams.AddMemberResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.TeamID != team.ID {
			t.Errorf("expected TeamID %d, got %d", team.ID, resp.TeamID)
		}
		if resp.UserID != user.ID {
			t.Errorf("expected UserID %d, got %d", user.ID, resp.UserID)
		}
	})

	t.Run("returns 404 for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		rec := doRequest(t, b, "999999", teams.AddMemberRequest{UserID: user.ID})

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for nonexistent user", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), teams.AddMemberRequest{UserID: 999999})

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("returns 409 on duplicate membership", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)
		team := dbfactory.Team(ctx, t, b.tx, nil)
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: user.ID})

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), teams.AddMemberRequest{UserID: user.ID})

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on missing user_id", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), struct{}{})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 400 on invalid team id", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, "abc", teams.AddMemberRequest{UserID: 1})

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListMembers(t *testing.T) {
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

	doRequest := func(t *testing.T, b *testBundle, teamID string, query string) *httptest.ResponseRecorder {
		t.Helper()
		url := "/teams/" + teamID + "/members"
		if query != "" {
			url += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := b.e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "id", Value: teamID}})
		if err := b.handler.ListMembers(c); err != nil {
			b.e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	t.Run("returns 200 with member list and pagination", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		alice := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Name: "Alice", Email: "alice@example.com", Score: 10})
		bob := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Name: "Bob", Email: "bob@example.com", Score: 5})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: alice.ID})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: bob.ID})

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), "page=1&page_size=20")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp teams.ListMembersResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 2 {
			t.Errorf("expected total=2, got %d", resp.Total)
		}
		if resp.Page != 1 {
			t.Errorf("expected page=1, got %d", resp.Page)
		}
		if resp.PageSize != 20 {
			t.Errorf("expected page_size=20, got %d", resp.PageSize)
		}
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 items in data, got %d", len(resp.Data))
		}
		if resp.Data[0].UserID != alice.ID {
			t.Errorf("expected first user_id=%d, got %d", alice.ID, resp.Data[0].UserID)
		}
		if resp.Data[0].Name != "Alice" {
			t.Errorf("expected name=Alice, got %s", resp.Data[0].Name)
		}
		if resp.Data[0].Email != "alice@example.com" {
			t.Errorf("expected email=alice@example.com, got %s", resp.Data[0].Email)
		}
		if resp.Data[0].Score != 10 {
			t.Errorf("expected score=10, got %d", resp.Data[0].Score)
		}
	})

	t.Run("returns 200 with empty array for team with no members", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), "")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Capture body before decoding so we can inspect raw JSON too.
		body := rec.Body.String()
		var resp teams.ListMembersResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Total != 0 {
			t.Errorf("expected total=0, got %d", resp.Total)
		}
		if resp.Data == nil {
			t.Error("expected data to be non-nil empty array, got null")
		}
		if len(resp.Data) != 0 {
			t.Errorf("expected empty data, got %d items", len(resp.Data))
		}

		// Ensure JSON encodes as [] not null
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

	t.Run("returns 400 on invalid team id", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, "abc", "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("uses defaults when page params omitted", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		rec := doRequest(t, b, fmt.Sprintf("%d", team.ID), "")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp teams.ListMembersResponse
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
