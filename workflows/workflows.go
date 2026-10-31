// Package workflows defines the four workflows of the reference and their steps:
//
//	IngestSource    Extract, Judge, Build, Score            one source, on its interval or on demand
//	IngestAll       Extract each source, Judge, Build, Score, then the BuildArtifacts steps
//	ConfirmLink     Apply                                   a team confirms or rejects an inferred link
//	DeclarePrimary  Apply                                   a data set owner's request, approved by a second person
//	BuildArtifacts  Snapshot, Diagrams, Insights             nightly, and at the end of IngestAll
//
// Every step is idempotent and passes nothing but the run id; what it produces goes to PostgreSQL.
package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/raskell-io/it-intelligence/artifacts"
	"github.com/raskell-io/it-intelligence/connectors"
	"github.com/raskell-io/it-intelligence/internal/config"
	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/resolver"
	"github.com/raskell-io/it-intelligence/resolver/provider"
	"github.com/raskell-io/it-intelligence/store"
	"github.com/raskell-io/it-intelligence/workflows/runner"
)

// Deps is what the steps need.
type Deps struct {
	Store    *store.Store
	Config   *config.Config
	Provider provider.Provider
	Log      *slog.Logger
}

// SourceInput is the input of IngestSource.
type SourceInput struct {
	Source string `json:"source"`
}

// LinkInput is the input of ConfirmLink.
type LinkInput struct {
	TargetType string `json:"target_type"` // edge or alias
	TargetID   string `json:"target_id"`
	Action     string `json:"action"` // confirm or reject
	Note       string `json:"note,omitempty"`
}

// PrimaryInput is the input of DeclarePrimary.
type PrimaryInput struct {
	Source    string `json:"source"`
	Dataset   string `json:"dataset"`
	Owner     string `json:"owner"`
	Requester string `json:"requester"`
	Approver  string `json:"approver"`
}

// All returns the workflows, ready for the runner.
func All(d *Deps) []runner.Workflow {
	modelSteps := []runner.Step{
		{Name: "Judge", Run: d.judge, Timeout: 30 * time.Minute},
		{Name: "Build", Run: d.build, Timeout: 10 * time.Minute},
		{Name: "Score", Run: d.score, Timeout: 10 * time.Minute},
	}
	artifactSteps := []runner.Step{
		{Name: "Snapshot", Run: d.snapshot},
		{Name: "Diagrams", Run: d.diagrams},
		{Name: "Insights", Run: d.insights},
	}
	var all []runner.Step
	for _, s := range d.Config.SourceNames() {
		src := s
		all = append(all, runner.Step{Name: "Extract " + src, Timeout: 30 * time.Minute,
			Run: func(ctx context.Context, r runner.Run) (any, error) { return d.extract(ctx, r.ID, src) }})
	}
	all = append(append(all, modelSteps...), artifactSteps...)

	return []runner.Workflow{
		{Name: "IngestSource", Steps: append([]runner.Step{{Name: "Extract", Timeout: 30 * time.Minute,
			Run: func(ctx context.Context, r runner.Run) (any, error) {
				var in SourceInput
				if err := json.Unmarshal(r.Input, &in); err != nil {
					return nil, err
				}
				return d.extract(ctx, r.ID, in.Source)
			}}}, modelSteps...)},
		{Name: "IngestAll", Steps: all},
		{Name: "ConfirmLink", Steps: append([]runner.Step{{Name: "Apply", Run: d.applyLink, Retry: runner.RetryPolicy{MaxAttempts: 1}}},
			runner.Step{Name: "Build", Run: d.build}, runner.Step{Name: "Score", Run: d.score})},
		{Name: "DeclarePrimary", Steps: []runner.Step{{Name: "Apply", Run: d.applyPrimary, Retry: runner.RetryPolicy{MaxAttempts: 1}},
			{Name: "Build", Run: d.build}}},
		{Name: "BuildArtifacts", Steps: artifactSteps},
	}
}

// Schedules returns one interval schedule per source and a nightly artifact build.
func Schedules(c *config.Config) []runner.Schedule {
	var out []runner.Schedule
	for _, name := range c.SourceNames() {
		if every := time.Duration(c.Sources[name].Every); every > 0 {
			out = append(out, runner.Schedule{Workflow: "IngestSource", Input: SourceInput{Source: name}, Every: every})
		}
	}
	return append(out, runner.Schedule{Workflow: "BuildArtifacts", Input: map[string]any{}, Every: 24 * time.Hour})
}

// ---- steps ----

func (d *Deps) extract(ctx context.Context, runID int64, source string) (any, error) {
	sc, ok := d.Config.Sources[source]
	if !ok {
		return nil, fmt.Errorf("no source %q in config/sources.yaml", source)
	}
	mode := sc.Mode
	if mode == "" {
		mode = "file"
	}
	conn, err := connectors.New(source, connectors.Config{Mode: mode, Dir: d.Config.Path(sc.Dir), Now: d.Config.Now(), Opts: sc.Opts})
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, ds := range conn.Datasets() {
		cycle := time.Duration(sc.Cycle)
		if cycle == 0 {
			cycle = 180 * 24 * time.Hour
		}
		if err := d.Store.RegisterDataset(ctx, source, ds, cycle); err != nil {
			return nil, err
		}
		var obs []connectors.Observation
		for o, err := range conn.Extract(ctx, ds, time.Time{}) {
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", source, ds, err)
			}
			obs = append(obs, o)
		}
		n, err := d.Store.WriteObservations(ctx, runID, source, ds, obs, d.Config.Now())
		if err != nil {
			return nil, err
		}
		out[ds] = map[string]int{"records": len(obs), "new_or_changed": n}
	}
	return out, nil
}

// Input gathers everything a build needs from the store.
func (d *Deps) Input(ctx context.Context) (resolver.Input, []connectors.Observation, error) {
	obs, err := d.Store.LatestObservations(ctx)
	if err != nil {
		return resolver.Input{}, nil, err
	}
	mctx := mapping.Context{Now: d.Config.Now(), Config: d.Config.Mapping}
	var as []mapping.Assertions
	for _, o := range obs {
		m, ok := mapping.For(o.Source)
		if !ok {
			return resolver.Input{}, nil, fmt.Errorf("no mapper for source %q", o.Source)
		}
		a, err := m.Map(mctx, o)
		if err != nil {
			d.Log.Warn("mapping failed", "source", o.Source, "dataset", o.Dataset, "id", o.ExternalID, "err", err)
			continue
		}
		as = append(as, a)
	}
	scores, err := d.Store.LatestScores(ctx)
	if err != nil {
		return resolver.Input{}, nil, err
	}
	js, err := d.Store.Judgements(ctx)
	if err != nil {
		return resolver.Input{}, nil, err
	}
	decs, err := d.Store.Decisions(ctx)
	if err != nil {
		return resolver.Input{}, nil, err
	}
	return resolver.Input{Assertions: as, SourceOrder: d.Config.SourceOrder, Scores: scores, Judgements: js,
		Decisions: decs, Config: d.Config.Resolver}, obs, nil
}

// judge asks the model about every candidate without a current judgement. It is the only step that
// calls a model, so a timed-out call retries without redoing extraction.
func (d *Deps) judge(ctx context.Context, _ runner.Run) (any, error) {
	in, _, err := d.Input(ctx)
	if err != nil {
		return nil, err
	}
	m := resolver.Build(in)
	asked := 0
	for _, c := range m.Candidates {
		if c.Judged {
			continue
		}
		j, err := d.Provider.Judge(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("judge %s: %w", c.Key, err)
		}
		if err := d.Store.SaveJudgement(ctx, j); err != nil {
			return nil, err
		}
		asked++
	}
	return map[string]any{"candidates": len(m.Candidates), "asked": asked, "provider": d.Provider.Name()}, nil
}

func (d *Deps) build(ctx context.Context, r runner.Run) (any, error) {
	in, _, err := d.Input(ctx)
	if err != nil {
		return nil, err
	}
	m := resolver.Build(in)
	if err := d.Store.WriteModel(ctx, r.ID, m); err != nil {
		return nil, err
	}
	pending := 0
	for _, e := range m.Edges {
		if e.Status == resolver.Pending {
			pending++
		}
	}
	return map[string]any{"entities": len(m.Entities), "edges": len(m.Edges), "aliases": len(m.Aliases),
		"pending": pending, "dropped": len(m.Dropped)}, nil
}

func (d *Deps) applyLink(ctx context.Context, r runner.Run) (any, error) {
	var in LinkInput
	if err := json.Unmarshal(r.Input, &in); err != nil {
		return nil, err
	}
	if r.Actor == "" {
		return nil, fmt.Errorf("a confirmation needs a person: no actor")
	}
	if in.Action != "confirm" && in.Action != "reject" {
		return nil, fmt.Errorf("action must be confirm or reject")
	}
	table := map[string]string{"edge": "edge", "alias": "alias"}[in.TargetType]
	if table == "" {
		return nil, fmt.Errorf("target_type must be edge or alias")
	}
	var basis string
	var ev []byte
	if err := d.Store.Pool.QueryRow(ctx, `select basis, evidence from `+table+` where id = $1`, in.TargetID).Scan(&basis, &ev); err != nil {
		return nil, fmt.Errorf("%s %s: %w", in.TargetType, in.TargetID, err)
	}
	var evidence struct {
		InputHash string `json:"input_hash"`
	}
	json.Unmarshal(ev, &evidence)
	if evidence.InputHash == "" {
		return nil, fmt.Errorf("%s %s was not inferred; only inferred links are confirmed or rejected", in.TargetType, in.TargetID)
	}
	// The log is append-only; a retried step must not log twice.
	var exists bool
	d.Store.Pool.QueryRow(ctx, `select exists(select 1 from decision_log where target_id = $1 and action = $2 and actor = $3 and evidence_hash = $4)`,
		in.TargetID, in.Action, r.Actor, evidence.InputHash).Scan(&exists)
	if !exists {
		if err := d.Store.LogDecision(ctx, in.TargetType, in.TargetID, in.Action, r.Actor, in.Note, evidence.InputHash); err != nil {
			return nil, err
		}
	}
	return map[string]any{"target": in.TargetID, "action": in.Action, "actor": r.Actor}, nil
}

func (d *Deps) applyPrimary(ctx context.Context, r runner.Run) (any, error) {
	var in PrimaryInput
	if err := json.Unmarshal(r.Input, &in); err != nil {
		return nil, err
	}
	if in.Requester == "" || in.Approver == "" || in.Requester == in.Approver {
		return nil, fmt.Errorf("declaring a data set primary needs a requester and a different approver")
	}
	if in.Owner == "" {
		return nil, fmt.Errorf("a primary data set needs an owner")
	}
	id := in.Source + "/" + in.Dataset
	tag, err := d.Store.Pool.Exec(ctx, `update dataset set standing = 'primary', owner = $3, declared_by = $4, declared_at = now()
		where source = $1 and dataset = $2`, in.Source, in.Dataset, in.Owner, in.Approver)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("no data set %s", id)
	}
	var exists bool
	d.Store.Pool.QueryRow(ctx, `select exists(select 1 from decision_log where target_id = $1 and action = 'approve_primary')`, id).Scan(&exists)
	if !exists {
		d.Store.LogDecision(ctx, "dataset", id, "request_primary", in.Requester, "owner "+in.Owner, "")
		d.Store.LogDecision(ctx, "dataset", id, "approve_primary", in.Approver, "owner "+in.Owner, "")
	}
	return map[string]any{"dataset": id, "standing": "primary", "owner": in.Owner}, nil
}

// ---- artifacts ----

func (d *Deps) snapshot(ctx context.Context, r runner.Run) (any, error) {
	return artifacts.Snapshot(ctx, d.Store, d.Config.Path("var/snapshots"), d.Config.Now())
}

func (d *Deps) diagrams(ctx context.Context, r runner.Run) (any, error) {
	return artifacts.Diagrams(ctx, d.Store, d.Config.Now())
}

func (d *Deps) insights(ctx context.Context, r runner.Run) (any, error) {
	return artifacts.Insights(ctx, d.Store, d.Config.Now())
}
