package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func TestJWTAuth(t *testing.T) {
	secret := []byte("secret")
	e := echo.New()
	mw := JWTAuth(secret)
	h := mw(func(c *echo.Context) error {
		return c.String(http.StatusOK, c.Get(ContextKeyUserID).(string))
	})

	t.Run("valid token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "123",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString(secret)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := h(c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "123" {
			t.Errorf("expected body %s, got %s", "123", rec.Body.String())
		}
	})

	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h(c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		he, ok := err.(*echo.HTTPError)
		if !ok || he.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %v", http.StatusUnauthorized, err)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Token 123")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h(c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		he, ok := err.(*echo.HTTPError)
		if !ok || he.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %v", http.StatusUnauthorized, err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h(c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		he, ok := err.(*echo.HTTPError)
		if !ok || he.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %v", http.StatusUnauthorized, err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "123",
			"exp": time.Now().Add(-time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString(secret)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h(c)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		he, ok := err.(*echo.HTTPError)
		if !ok || he.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %v", http.StatusUnauthorized, err)
		}
	})
}
