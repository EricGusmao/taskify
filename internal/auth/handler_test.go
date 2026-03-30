package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EricGusmao/taskify/internal/auth"
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

func TestHandler_Register(t *testing.T) {
	type testBundle struct {
		handler *auth.Handler
		e       *echo.Echo
		tx      *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		svc := auth.NewService(auth.NewUserRepository(tx), zap.NewNop())
		return &testBundle{
			handler: auth.NewHandler(svc),
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
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := b.e.NewContext(req, rec)
		_ = b.handler.Register(c)
		return rec
	}

	t.Run("returns 201 with user data", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, auth.RegisterRequest{
			Name:     "Alice",
			Email:    "alice@example.com",
			Password: "securepassword",
		})

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		var resp auth.RegisterResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.ID == 0 {
			t.Error("expected ID > 0")
		}
		if resp.Email != "alice@example.com" {
			t.Errorf("expected email alice@example.com, got %s", resp.Email)
		}
		if resp.Name != "Alice" {
			t.Errorf("expected name Alice, got %s", resp.Name)
		}
	})

	t.Run("returns 409 on duplicate email", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{
			Email: "dup@example.com",
		})

		rec := doRequest(t, b, auth.RegisterRequest{
			Name:     "Bob",
			Email:    "dup@example.com",
			Password: "securepassword",
		})

		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on missing fields", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, struct{}{})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on invalid email", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, auth.RegisterRequest{
			Name:     "Dan",
			Email:    "not-an-email",
			Password: "securepassword",
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("returns 422 on short password", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, auth.RegisterRequest{
			Name:     "Eve",
			Email:    "eve@example.com",
			Password: "short",
		})

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rec.Code)
		}
	})

	t.Run("does not leak password hash", func(t *testing.T) {
		t.Parallel()
		b, _ := setup(t)

		rec := doRequest(t, b, auth.RegisterRequest{
			Name:     "Frank",
			Email:    "frank@example.com",
			Password: "securepassword",
		})

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}

		var raw map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&raw); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if _, ok := raw["password_hash"]; ok {
			t.Error("response must not contain password_hash")
		}
	})
}
