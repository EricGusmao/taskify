package teams_test

import (
	"context"
	"errors"
	"testing"

	"github.com/EricGusmao/taskify/internal/teams"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestService_ListMembers(t *testing.T) {
	type testBundle struct {
		svc *teams.Service
		tx  *gorm.DB
	}

	setup := func(t *testing.T) (*testBundle, context.Context) {
		t.Helper()
		db := testhelper.NewMySQLContainer(t)
		tx := testhelper.TestTx(t, db)
		svc := teams.NewService(teams.NewRepository(tx), teams.NewUserRepository(tx), zap.NewNop())
		return &testBundle{svc: svc, tx: tx}, context.Background()
	}

	t.Run("returns members with scores", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		alice := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Name: "Alice", Email: "alice@example.com", Score: 10})
		bob := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Name: "Bob", Email: "bob@example.com", Score: 5})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: alice.ID})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: bob.ID})

		page, err := b.svc.ListMembers(ctx, teams.ListMembersInput{
			TeamID:   team.ID,
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Total != 2 {
			t.Errorf("expected total=2, got %d", page.Total)
		}
		if len(page.Data) != 2 {
			t.Fatalf("expected 2 members in data, got %d", len(page.Data))
		}

		// results are ordered by score DESC: alice first
		if page.Data[0].UserID != alice.ID {
			t.Errorf("expected first member user_id=%d (alice), got %d", alice.ID, page.Data[0].UserID)
		}
		if page.Data[0].Name != "Alice" {
			t.Errorf("expected name=Alice, got %s", page.Data[0].Name)
		}
		if page.Data[0].Email != "alice@example.com" {
			t.Errorf("expected email=alice@example.com, got %s", page.Data[0].Email)
		}
		if page.Data[0].Score != 10 {
			t.Errorf("expected score=10, got %d", page.Data[0].Score)
		}
		if page.Data[1].UserID != bob.ID {
			t.Errorf("expected second member user_id=%d (bob), got %d", bob.ID, page.Data[1].UserID)
		}
	})

	t.Run("returns empty slice for team with no members", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		page, err := b.svc.ListMembers(ctx, teams.ListMembersInput{
			TeamID:   team.ID,
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Total != 0 {
			t.Errorf("expected total=0, got %d", page.Total)
		}
		if page.Data == nil {
			t.Error("expected data to be non-nil empty slice, got nil")
		}
		if len(page.Data) != 0 {
			t.Errorf("expected empty data, got %d items", len(page.Data))
		}
	})

	t.Run("returns ErrTeamNotFound for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		_, err := b.svc.ListMembers(ctx, teams.ListMembersInput{
			TeamID:   999999,
			Page:     1,
			PageSize: 20,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, teams.ErrTeamNotFound) {
			t.Errorf("expected ErrTeamNotFound, got %v", err)
		}
	})

	t.Run("respects page and page_size", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		for i := 0; i < 5; i++ {
			u := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: i})
			dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u.ID})
		}

		page, err := b.svc.ListMembers(ctx, teams.ListMembersInput{
			TeamID:   team.ID,
			Page:     2,
			PageSize: 2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Total != 5 {
			t.Errorf("expected total=5, got %d", page.Total)
		}
		if len(page.Data) != 2 {
			t.Errorf("expected 2 results on page 2, got %d", len(page.Data))
		}
		if page.Page != 2 {
			t.Errorf("expected page=2, got %d", page.Page)
		}
		if page.PageSize != 2 {
			t.Errorf("expected page_size=2, got %d", page.PageSize)
		}
	})
}
