package auth_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/EricGusmao/taskify/internal/auth"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var testJWTSecret = []byte("test-jwt-secret-that-is-at-least-64-characters-long-for-testing!")

func TestService_Login(t *testing.T) {
	type testBundle struct {
		svc *auth.Service
		tx  *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		return &testBundle{
			svc: auth.NewService(auth.NewUserRepository(tx), zap.NewNop(), testJWTSecret),
			tx:  tx,
		}, context.Background()
	}

	t.Run("returns token for valid credentials", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		token, err := b.svc.Login(ctx, auth.LoginInput{
			Email:    user.Email,
			Password: "password",
		})
		if err != nil {
			t.Fatalf("Login: unexpected error: %v", err)
		}
		if token == "" {
			t.Fatal("expected non-empty token")
		}

		parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
			return testJWTSecret, nil
		})
		if err != nil {
			t.Fatalf("jwt.ParseWithClaims: %v", err)
		}

		claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
		if !ok || !parsed.Valid {
			t.Fatal("expected valid parsed claims")
		}

		expectedSub := strconv.FormatUint(uint64(user.ID), 10)
		if claims.Subject != expectedSub {
			t.Errorf("expected subject %s, got %s", expectedSub, claims.Subject)
		}
	})

	t.Run("rejects unknown email", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		_, err := b.svc.Login(ctx, auth.LoginInput{
			Email:    "nobody@example.com",
			Password: "password",
		})
		if !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("rejects wrong password", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user := dbfactory.User(ctx, t, b.tx, nil)

		_, err := b.svc.Login(ctx, auth.LoginInput{
			Email:    user.Email,
			Password: "wrongpassword",
		})
		if !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}
