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

func TestService_GetRanking(t *testing.T) {
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

	t.Run("returns members ordered by score descending", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		u1 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 5})
		u2 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 20})
		u3 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 10})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u1.ID})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u2.ID})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u3.ID})

		page, err := b.svc.GetRanking(ctx, teams.RankingInput{TeamID: team.ID, Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Total != 3 {
			t.Errorf("expected total=3, got %d", page.Total)
		}
		if len(page.Data) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(page.Data))
		}

		expected := []struct {
			score int
			rank  int
		}{
			{20, 1},
			{10, 2},
			{5, 3},
		}
		for i, want := range expected {
			if page.Data[i].Score != want.score {
				t.Errorf("entry[%d]: expected score=%d, got %d", i, want.score, page.Data[i].Score)
			}
			if page.Data[i].Rank != want.rank {
				t.Errorf("entry[%d]: expected rank=%d, got %d", i, want.rank, page.Data[i].Rank)
			}
		}
	})

	t.Run("assigns correct ranks on page 2", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		for i, score := range []int{50, 40, 30, 20, 10} {
			u := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: score})
			dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u.ID})
			_ = i
		}

		page, err := b.svc.GetRanking(ctx, teams.RankingInput{TeamID: team.ID, Page: 2, PageSize: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Total != 5 {
			t.Errorf("expected total=5, got %d", page.Total)
		}
		if len(page.Data) != 2 {
			t.Fatalf("expected 2 entries on page 2, got %d", len(page.Data))
		}
		if page.Data[0].Rank != 3 {
			t.Errorf("expected rank=3 for first entry on page 2, got %d", page.Data[0].Rank)
		}
		if page.Data[1].Rank != 4 {
			t.Errorf("expected rank=4 for second entry on page 2, got %d", page.Data[1].Rank)
		}
	})

	t.Run("returns empty slice for team with no members", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)

		page, err := b.svc.GetRanking(ctx, teams.RankingInput{TeamID: team.ID, Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 0 {
			t.Errorf("expected total=0, got %d", page.Total)
		}
		if page.Data == nil {
			t.Error("expected non-nil empty slice, got nil")
		}
		if len(page.Data) != 0 {
			t.Errorf("expected 0 entries, got %d", len(page.Data))
		}
	})

	t.Run("returns ErrTeamNotFound for nonexistent team", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		_, err := b.svc.GetRanking(ctx, teams.RankingInput{TeamID: 999999, Page: 1, PageSize: 20})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, teams.ErrTeamNotFound) {
			t.Errorf("expected ErrTeamNotFound, got %v", err)
		}
	})

	t.Run("breaks ties deterministically by user ID", func(t *testing.T) {
		t.Parallel()
		b, ctx := setup(t)

		team := dbfactory.Team(ctx, t, b.tx, nil)
		u1 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 15})
		u2 := dbfactory.User(ctx, t, b.tx, &dbfactory.UserOpts{Score: 15})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u1.ID})
		dbfactory.Member(ctx, t, b.tx, &dbfactory.MemberOpts{TeamID: team.ID, UserID: u2.ID})

		page, err := b.svc.GetRanking(ctx, teams.RankingInput{TeamID: team.ID, Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Data) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(page.Data))
		}
		// Secondary sort is by user ID ASC — u1 has a lower ID so it comes first.
		if page.Data[0].UserID != u1.ID {
			t.Errorf("expected first user_id=%d (lower ID), got %d", u1.ID, page.Data[0].UserID)
		}
		if page.Data[1].UserID != u2.ID {
			t.Errorf("expected second user_id=%d (higher ID), got %d", u2.ID, page.Data[1].UserID)
		}
	})
}
