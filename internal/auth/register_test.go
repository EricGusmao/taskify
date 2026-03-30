package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EricGusmao/taskify/internal/auth"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestService_Register(t *testing.T) {
	type testBundle struct {
		svc *auth.Service
		tx  *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		return &testBundle{
			svc: auth.NewService(auth.NewUserRepository(tx), zap.NewNop(), []byte("test-jwt-secret-that-is-at-least-64-characters-long-for-testing!")),
			tx:  tx,
		}, context.Background()
	}

	t.Run("creates user with hashed password", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user, err := b.svc.Register(ctx, auth.RegisterInput{
			Name:     "Alice",
			Email:    "alice@example.com",
			Password: "securepassword",
		})
		if err != nil {
			t.Fatalf("Register: unexpected error: %v", err)
		}
		if user.ID == 0 {
			t.Error("expected user.ID > 0")
		}
		if user.Score != 0 {
			t.Errorf("expected score 0, got %d", user.Score)
		}
		if user.Email != "alice@example.com" {
			t.Errorf("expected email alice@example.com, got %s", user.Email)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("securepassword")); err != nil {
			t.Errorf("password hash mismatch: %v", err)
		}
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{
			Email: "dup@example.com",
		})

		_, err := b.svc.Register(ctx, auth.RegisterInput{
			Name:     "Bob",
			Email:    "dup@example.com",
			Password: "securepassword",
		})
		if !errors.Is(err, auth.ErrEmailTaken) {
			t.Errorf("expected ErrEmailTaken, got %v", err)
		}
	})

	t.Run("hashes password with correct cost", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		user, err := b.svc.Register(ctx, auth.RegisterInput{
			Name:     "Carol",
			Email:    "carol@example.com",
			Password: "securepassword",
		})
		if err != nil {
			t.Fatalf("Register: unexpected error: %v", err)
		}

		cost, err := bcrypt.Cost([]byte(user.PasswordHash))
		if err != nil {
			t.Fatalf("bcrypt.Cost: %v", err)
		}
		if cost != 12 {
			t.Errorf("expected bcrypt cost 12, got %d", cost)
		}
	})
}
