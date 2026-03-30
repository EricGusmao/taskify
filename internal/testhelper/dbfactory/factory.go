// Package dbfactory provides test data factories for integration tests.
package dbfactory

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/EricGusmao/taskify/internal/auth"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var seq atomic.Int64

func seqStr() string { return fmt.Sprintf("%06d", seq.Add(1)) }

// UserOpts configures the User factory.
// Fields with no default generation are marked required — leave others empty to use generated values.
type UserOpts struct {
	Name         string
	Email        string `validate:"omitempty,email"`
	PasswordHash string
	Score        int
}

// User inserts an auth.User into the database and returns it.
// Calls t.Fatal on any error.
func User(ctx context.Context, t *testing.T, tx *gorm.DB, opts *UserOpts) *auth.User {
	t.Helper()

	if opts == nil {
		opts = &UserOpts{}
	}

	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("User %s", seqStr())
	}

	email := opts.Email
	if email == "" {
		email = fmt.Sprintf("user-%s@example.com", seqStr())
	}

	hash := opts.PasswordHash
	if hash == "" {
		b, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("dbfactory.User: hash password: %v", err)
		}
		hash = string(b)
	}

	u := &auth.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Score:        opts.Score,
	}

	if err := tx.WithContext(ctx).Create(u).Error; err != nil {
		t.Fatalf("dbfactory.User: %v", err)
	}

	return u
}
