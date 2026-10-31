// Package runner runs workflows: fixed lists of idempotent steps, with their state in PostgreSQL.
//
// It is deliberately small. A run is claimed with SELECT ... FOR UPDATE SKIP LOCKED, so several
// processes can share the work and a run never executes twice at once. Every step attempt is recorded.
// After a crash, a run resumes at the first step that did not succeed. Steps pass a run id, never data:
// whatever a step produces it writes to the database, which is what makes resuming trivial.
//
// When workflows start waiting on people or events for days, or fan out widely, a durable execution
// engine (Temporal, Restate) earns its place. Steps here already have the shape of their activities.
package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run is what a step gets to know about the run it belongs to.
type Run struct {
	ID       int64
	Workflow string
	Input    json.RawMessage
	Actor    string
}

// Step is one idempotent unit of work. Its output (counts, ids) is stored for the runs page.
type Step struct {
	Name    string
	Run     func(ctx context.Context, r Run) (any, error)
	Timeout time.Duration
	Retry   RetryPolicy
}

// RetryPolicy is exponential backoff with jitter.
type RetryPolicy struct {
	MaxAttempts int
	Initial     time.Duration
	Max         time.Duration
}

// DefaultRetry retries five times, from one second up to a minute.
var DefaultRetry = RetryPolicy{MaxAttempts: 5, Initial: time.Second, Max: time.Minute}

// Workflow is a named list of steps.
type Workflow struct {
	Name  string
	Steps []Step
}

// Schedule enqueues a workflow when its last successful run with the same input is older than Every.
type Schedule struct {
	Workflow string
	Input    any
	Every    time.Duration
}

// Runner executes runs.
type Runner struct {
	pool       *pgxpool.Pool
	workflows  map[string]Workflow
	schedules  []Schedule
	id         string
	Poll       time.Duration
	StaleAfter time.Duration
	log        *slog.Logger
}

// New builds a runner.
func New(pool *pgxpool.Pool, log *slog.Logger, wfs ...Workflow) *Runner {
	host, _ := os.Hostname()
	r := &Runner{pool: pool, workflows: map[string]Workflow{}, id: fmt.Sprintf("%s-%d", host, os.Getpid()),
		Poll: time.Second, StaleAfter: 2 * time.Minute, log: log}
	for _, w := range wfs {
		r.workflows[w.Name] = w
	}
	return r
}

// AddSchedule registers a schedule.
func (r *Runner) AddSchedule(s Schedule) { r.schedules = append(r.schedules, s) }

// Enqueue starts a run.
func (r *Runner) Enqueue(ctx context.Context, workflow string, input any, actor string) (int64, error) {
	if _, ok := r.workflows[workflow]; !ok {
		return 0, fmt.Errorf("no workflow %q", workflow)
	}
	b, err := json.Marshal(input)
	if err != nil {
		return 0, err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `insert into run (workflow, input, actor) values ($1, $2, nullif($3, '')) returning id`, workflow, b, actor).Scan(&id)
	return id, err
}

// Wait blocks until a run has finished and returns its status.
func (r *Runner) Wait(ctx context.Context, id int64) (string, error) {
	for {
		var status string
		var msg *string
		if err := r.pool.QueryRow(ctx, `select status, error from run where id = $1`, id).Scan(&status, &msg); err != nil {
			return "", err
		}
		if status == "succeeded" {
			return status, nil
		}
		if status == "failed" {
			m := ""
			if msg != nil {
				m = *msg
			}
			return status, fmt.Errorf("run %d failed: %s", id, m)
		}
		select {
		case <-ctx.Done():
			return status, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// Start runs the claim loop and the scheduler until ctx is cancelled.
func (r *Runner) Start(ctx context.Context) {
	go r.scheduleLoop(ctx)
	for {
		claimed, err := r.claimAndRun(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			r.log.Error("runner", "err", err)
		}
		if claimed {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.Poll):
		}
	}
}

func (r *Runner) claimAndRun(ctx context.Context) (bool, error) {
	var run Run
	var actor *string
	err := r.pool.QueryRow(ctx, `
		update run set status = 'running', claimed_by = $1, heartbeat_at = now(), started_at = coalesce(started_at, now())
		where id = (
			select id from run
			where (status = 'queued' and not_before <= now())
			   or (status = 'running' and heartbeat_at < now() - make_interval(secs => $2))
			order by id for update skip locked limit 1)
		returning id, workflow, input, actor`, r.id, r.StaleAfter.Seconds()).Scan(&run.ID, &run.Workflow, &run.Input, &actor)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if actor != nil {
		run.Actor = *actor
	}
	hbCtx, stop := context.WithCancel(ctx)
	go r.heartbeat(hbCtx, run.ID)
	err = r.execute(ctx, run)
	stop()
	status, msg := "succeeded", ""
	if err != nil {
		status, msg = "failed", err.Error()
		r.log.Error("run failed", "run", run.ID, "workflow", run.Workflow, "err", err)
	} else {
		r.log.Info("run succeeded", "run", run.ID, "workflow", run.Workflow)
	}
	// A cancelled context means we are shutting down: leave the run claimed, the heartbeat goes stale
	// and the next runner resumes it.
	if errors.Is(err, context.Canceled) {
		return true, nil
	}
	_, uerr := r.pool.Exec(context.Background(), `update run set status = $2, error = nullif($3, ''), finished_at = now() where id = $1`, run.ID, status, msg)
	return true, uerr
}

func (r *Runner) heartbeat(ctx context.Context, id int64) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.pool.Exec(context.Background(), `update run set heartbeat_at = now() where id = $1`, id)
		}
	}
}

func (r *Runner) execute(ctx context.Context, run Run) error {
	wf, ok := r.workflows[run.Workflow]
	if !ok {
		return fmt.Errorf("no workflow %q", run.Workflow)
	}
	for pos, step := range wf.Steps {
		var done bool
		var attempts int
		if err := r.pool.QueryRow(ctx, `select coalesce(bool_or(status = 'succeeded'), false), count(*) from run_step where run_id = $1 and step = $2`,
			run.ID, step.Name).Scan(&done, &attempts); err != nil {
			return err
		}
		if done {
			continue // resumed run: this step already succeeded
		}
		policy := step.Retry
		if policy.MaxAttempts == 0 {
			policy = DefaultRetry
		}
		for {
			attempts++
			if err := r.attempt(ctx, run, pos, step, attempts); err == nil {
				break
			} else if errors.Is(err, context.Canceled) || attempts >= policy.MaxAttempts {
				return fmt.Errorf("step %s: %w", step.Name, err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff(policy, attempts)):
			}
		}
	}
	return nil
}

func backoff(p RetryPolicy, attempt int) time.Duration {
	d := p.Initial << (attempt - 1)
	if d > p.Max || d <= 0 {
		d = p.Max
	}
	return d/2 + time.Duration(rand.Int64N(int64(d/2)+1))
}

func (r *Runner) attempt(ctx context.Context, run Run, pos int, step Step, attempt int) error {
	if _, err := r.pool.Exec(ctx, `insert into run_step (run_id, step, position, attempt, status, started_at) values ($1, $2, $3, $4, 'running', now())`,
		run.ID, step.Name, pos, attempt); err != nil {
		return err
	}
	sctx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		sctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}
	out, err := safeRun(sctx, run, step)
	status, msg := "succeeded", ""
	if err != nil {
		status, msg = "failed", err.Error()
	}
	b, _ := json.Marshal(out)
	if b == nil || string(b) == "null" {
		b = []byte("{}")
	}
	if _, uerr := r.pool.Exec(context.Background(), `update run_step set status = $4, error = nullif($5, ''), output = $6, finished_at = now()
		where run_id = $1 and step = $2 and attempt = $3`, run.ID, step.Name, attempt, status, msg, b); uerr != nil {
		return uerr
	}
	return err
}

func safeRun(ctx context.Context, run Run, step Step) (out any, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return step.Run(ctx, run)
}

func (r *Runner) scheduleLoop(ctx context.Context) {
	if len(r.schedules) == 0 {
		return
	}
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		for _, s := range r.schedules {
			b, _ := json.Marshal(s.Input)
			var due bool
			err := r.pool.QueryRow(ctx, `
				select not exists (select 1 from run where workflow = $1 and input = $2 and status in ('queued', 'running'))
				   and coalesce((select max(finished_at) from run where workflow = $1 and input = $2 and status = 'succeeded') < now() - make_interval(secs => $3), true)`,
				s.Workflow, b, s.Every.Seconds()).Scan(&due)
			if err == nil && due {
				if _, err := r.Enqueue(ctx, s.Workflow, s.Input, "schedule"); err == nil {
					r.log.Info("scheduled", "workflow", s.Workflow, "input", string(b))
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
