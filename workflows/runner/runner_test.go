package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raskell-io/it-intelligence/store"
)

// These tests need PostgreSQL: ITX_TEST_DATABASE_URL, for example the dev database of `make dev-db`.
func testPool(t *testing.T) *pgxpool.Pool {
	url := os.Getenv("ITX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("ITX_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := store.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s.Pool
}

func TestRetryAndResume(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	var firstRuns, flakyCalls, lastRuns atomic.Int32
	quick := RetryPolicy{MaxAttempts: 3, Initial: 10 * time.Millisecond, Max: 20 * time.Millisecond}
	wf := Workflow{Name: "test-retry-" + time.Now().Format("150405.000"), Steps: []Step{
		{Name: "first", Run: func(context.Context, Run) (any, error) { firstRuns.Add(1); return nil, nil }},
		{Name: "flaky", Retry: quick, Run: func(context.Context, Run) (any, error) {
			if flakyCalls.Add(1) < 3 {
				return nil, errors.New("the source timed out")
			}
			return map[string]int{"records": 7}, nil
		}},
		{Name: "last", Run: func(context.Context, Run) (any, error) { lastRuns.Add(1); return nil, nil }},
	}}
	r := New(pool, log, wf)
	r.Poll = 20 * time.Millisecond

	// Pretend an earlier process already finished "first" and then crashed holding the claim.
	id, err := r.Enqueue(ctx, wf.Name, map[string]any{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into run_step (run_id, step, position, attempt, status, started_at, finished_at) values ($1, 'first', 0, 1, 'succeeded', now(), now())`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `update run set status = 'running', claimed_by = 'dead', heartbeat_at = now() - interval '1 hour' where id = $1`, id); err != nil {
		t.Fatal(err)
	}

	go r.Start(ctx)
	if _, err := r.Wait(ctx, id); err != nil {
		t.Fatal(err)
	}
	if firstRuns.Load() != 0 {
		t.Errorf("a succeeded step ran again after resume")
	}
	if flakyCalls.Load() != 3 || lastRuns.Load() != 1 {
		t.Errorf("flaky ran %d times, last %d times", flakyCalls.Load(), lastRuns.Load())
	}
	var attempts int
	pool.QueryRow(ctx, `select count(*) from run_step where run_id = $1 and step = 'flaky'`, id).Scan(&attempts)
	if attempts != 3 {
		t.Errorf("recorded %d attempts of flaky, want 3", attempts)
	}
}
