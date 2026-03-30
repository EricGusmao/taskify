// Package testhelper provides shared test utilities for integration tests.
package testhelper

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/EricGusmao/taskify/internal/auth"
	"github.com/EricGusmao/taskify/internal/infra"
	"github.com/EricGusmao/taskify/internal/teams"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"gorm.io/gorm"
)

var (
	once     sync.Once
	sharedDB *gorm.DB
	setupErr error
)

// NewMySQLContainer returns a *gorm.DB backed by a MySQL testcontainer.
// The container starts once per binary run; subsequent calls return the same instance.
// Calls t.Skip if Docker is unavailable.
func NewMySQLContainer(t testing.TB) *gorm.DB {
	t.Helper()

	once.Do(func() {
		ctx := context.Background()

		container, err := mysql.Run(
			ctx,
			"mysql:8",
			mysql.WithDatabase("taskify_test"),
			mysql.WithUsername("root"),
			mysql.WithPassword("root"),
		)
		if err != nil {
			setupErr = err
			return
		}

		dsn, err := container.ConnectionString(ctx, "parseTime=true&charset=utf8mb4")
		if err != nil {
			setupErr = err
			return
		}

		db, err := infra.NewDB(dsn)
		if err != nil {
			setupErr = err
			return
		}

		if err := db.AutoMigrate(&auth.User{}, &teams.Team{}); err != nil {
			setupErr = err
			return
		}

		sharedDB = db
	})

	if setupErr != nil {
		t.Skip("testhelper: skipping: docker not available or container failed:", setupErr)
	}

	return sharedDB
}

// TestTx begins a *gorm.DB transaction and registers its rollback via t.Cleanup.
// Each test that calls TestTx gets a fully isolated transaction.
func TestTx(t testing.TB, db *gorm.DB) *gorm.DB {
	t.Helper()

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("testhelper: begin tx: %v", tx.Error)
	}

	t.Cleanup(func() {
		if err := tx.Rollback().Error; err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("testhelper: rollback tx: %v", err)
		}
	})

	return tx
}
