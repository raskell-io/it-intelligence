// Command itx is the one binary of it-intelligence.
//
//	itx serve      REST on /api/v1, MCP on /mcp, the workflow runner, health on /healthz
//	itx migrate    apply the vocabulary and the store migrations
//	itx seed       register the configured sources and their data sets
//	itx ingest     run IngestAll (or IngestSource for one source) and wait for it
//	itx verify     check the resolver against the answer key and the article's claims
//	itx runs       list workflow runs, or show one
//	itx resolver   record fixtures from a real model
//	itx snapshot   list snapshots, or diff two
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/raskell-io/it-intelligence/api/mcp"
	"github.com/raskell-io/it-intelligence/api/query"
	"github.com/raskell-io/it-intelligence/api/rest"
	_ "github.com/raskell-io/it-intelligence/connectors/cluster"
	_ "github.com/raskell-io/it-intelligence/connectors/earepo"
	_ "github.com/raskell-io/it-intelligence/connectors/gateway"
	_ "github.com/raskell-io/it-intelligence/connectors/githost"
	_ "github.com/raskell-io/it-intelligence/connectors/itsm"
	_ "github.com/raskell-io/it-intelligence/connectors/waf"
	"github.com/raskell-io/it-intelligence/internal/config"
	"github.com/raskell-io/it-intelligence/internal/verify"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
	"github.com/raskell-io/it-intelligence/resolver"
	"github.com/raskell-io/it-intelligence/resolver/provider"
	"github.com/raskell-io/it-intelligence/store"
	"github.com/raskell-io/it-intelligence/workflows"
	"github.com/raskell-io/it-intelligence/workflows/runner"
)

const usage = `itx: the it-intelligence binary

  itx serve [-addr :8080] [-schedules]   REST, MCP and the workflow runner
  itx migrate                             apply the vocabulary and migrations
  itx seed                                register the configured data sets
  itx ingest [-source NAME] [-inline]     ingest everything (or one source) and wait
  itx verify [-resolver] [-article]       check against seed/truth.yaml and the article
  itx runs [ID]                           list runs, or show one with its steps
  itx resolver record [-provider KIND] [-model ID] [-base-url URL]
                                          judge every candidate with a real model, rewrite seed/judgements.jsonl
  itx snapshot list | diff A B            snapshots of the model

Environment: ITX_DATABASE_URL (default postgres://itx:itx@localhost:55432/itx), ITX_ROOT (default .)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	time.Local = time.UTC // timestamps in answers are UTC, whatever the host says
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, log, os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "itx:", err)
		os.Exit(1)
	}
}

type env struct {
	cfg   *config.Config
	store *store.Store
	q     *query.Q
	log   *slog.Logger
}

func open(ctx context.Context, log *slog.Logger) (*env, error) {
	root := getenv("ITX_ROOT", ".")
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	s, err := store.Open(ctx, getenv("ITX_DATABASE_URL", "postgres://itx:itx@localhost:55432/itx"))
	if err != nil {
		return nil, err
	}
	if err := s.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &env{cfg: cfg, store: s, q: &query.Q{Pool: s.Pool}, log: log}, nil
}

func (e *env) runner(prov provider.Provider) *runner.Runner {
	d := &workflows.Deps{Store: e.store, Config: e.cfg, Provider: prov, Log: e.log}
	return runner.New(e.store.Pool, e.log, workflows.All(d)...)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func run(ctx context.Context, log *slog.Logger, cmd string, args []string) error {
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "migrate":
		e, err := open(ctx, log)
		if err != nil {
			return err
		}
		defer e.store.Close()
		fmt.Println("migrated; Context Model", vocab.Version)
		return nil
	case "seed":
		return seed(ctx, log)
	case "serve":
		return serve(ctx, log, args)
	case "ingest":
		return ingest(ctx, log, args)
	case "verify":
		return verifyCmd(ctx, log, args)
	case "runs":
		return runs(ctx, log, args)
	case "resolver":
		return resolverCmd(ctx, log, args)
	case "snapshot":
		return snapshotCmd(ctx, log, args)
	}
	return fmt.Errorf("unknown command %q\n%s", cmd, usage)
}

func seed(ctx context.Context, log *slog.Logger) error {
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	for _, name := range e.cfg.SourceNames() {
		sc := e.cfg.Sources[name]
		fmt.Printf("  %-8s %s mode, %s\n", name, sc.Mode, sc.Dir)
	}
	fmt.Println("sources registered; data sets appear on their first extract. Run `itx ingest`.")
	return nil
}

func serve(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", getenv("ITX_ADDR", ":8080"), "listen address")
	schedules := fs.Bool("schedules", os.Getenv("ITX_SCHEDULES") == "1", "ingest every source on its interval")
	noRunner := fs.Bool("no-runner", false, "do not run workflows in this process")
	fs.Parse(args)
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	prov, err := provider.New(e.cfg.Provider)
	if err != nil {
		return err
	}
	r := e.runner(prov)
	if *schedules {
		for _, s := range workflows.Schedules(e.cfg) {
			r.AddSchedule(s)
		}
	}
	if !*noRunner {
		go r.Start(ctx)
	}
	people, err := rest.LoadPeople(e.cfg.Path("config/people.yaml"))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	(&rest.API{Q: e.q, Runs: r, People: people, Now: e.cfg.Now}).Routes(mux)
	mux.Handle("/mcp", mcp.Handler(mcp.Server(e.q, vocab.Version)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := e.store.Pool.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	srv := &http.Server{Addr: *addr, Handler: logRequests(log, mux), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Info("serving", "addr", *addr, "rest", "/api/v1", "mcp", "/mcp", "provider", prov.Name(), "schedules", *schedules)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func logRequests(log *slog.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		if r.URL.Path != "/healthz" {
			log.Debug("http", "method", r.Method, "path", r.URL.Path, "took", time.Since(start))
		}
	})
}

func ingest(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	source := fs.String("source", "", "ingest one source only")
	inline := fs.Bool("inline", false, "run the workflow in this process instead of waiting for `itx serve`")
	fs.Bool("all", true, "ingest every configured source (the default)")
	fs.Bool("wait", true, "wait for the run to finish (the default)")
	fs.Parse(args)
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	prov, err := provider.New(e.cfg.Provider)
	if err != nil {
		return err
	}
	r := e.runner(prov)
	wf, in := "IngestAll", any(map[string]any{})
	if *source != "" {
		wf, in = "IngestSource", workflows.SourceInput{Source: *source}
	}
	id, err := r.Enqueue(ctx, wf, in, "")
	if err != nil {
		return err
	}
	if *inline {
		rctx, cancel := context.WithCancel(ctx)
		defer cancel()
		go r.Start(rctx)
	}
	fmt.Printf("run %d: %s\n", id, wf)
	status, werr := r.Wait(ctx, id)
	printRun(ctx, e.q, id)
	if werr != nil {
		return werr
	}
	fmt.Println(status)
	return nil
}

func printRun(ctx context.Context, q *query.Q, id int64) {
	runs, err := q.Runs(ctx, id, 1)
	if err != nil || len(runs) == 0 {
		return
	}
	for _, s := range runs[0].Steps {
		out, _ := json.Marshal(s["output"])
		line := fmt.Sprintf("  %-16v %-9v %6vms  %s", s["step"], s["status"], s["duration_ms"], out)
		if e, ok := s["error"]; ok {
			line += "  error: " + fmt.Sprint(e)
		}
		fmt.Println(line)
	}
}

func verifyCmd(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	onlyResolver := fs.Bool("resolver", false, "only the resolver checks")
	onlyArticle := fs.Bool("article", false, "only the article checks")
	fs.Parse(args)
	both := !*onlyResolver && !*onlyArticle
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	rep, err := verify.Run(ctx, e.q, e.cfg.Path("seed/truth.yaml"), e.cfg.Path("config/verify.yaml"), both || *onlyResolver, both || *onlyArticle, os.Stdout)
	if err != nil {
		return err
	}
	if rep.Failed() {
		return fmt.Errorf("verify failed")
	}
	fmt.Println("all checks pass")
	return nil
}

func runs(ctx context.Context, log *slog.Logger, args []string) error {
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	if len(args) == 1 {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		printRun(ctx, e.q, id)
		return nil
	}
	rs, err := e.q.Runs(ctx, 0, 20)
	if err != nil {
		return err
	}
	for _, r := range rs {
		fmt.Printf("%5d  %-15s %-10s %s  %s\n", r.ID, r.Workflow, r.Status, r.CreatedAt.Format(time.DateTime), string(r.Input))
	}
	return nil
}

func resolverCmd(ctx context.Context, log *slog.Logger, args []string) error {
	if len(args) == 0 || args[0] != "record" {
		return fmt.Errorf("usage: itx resolver record [-provider KIND] [-model ID] [-base-url URL]")
	}
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	kind := fs.String("provider", "anthropic", "anthropic or openai-compatible")
	model := fs.String("model", "", "model id")
	base := fs.String("base-url", "", "openai-compatible base URL")
	out := fs.String("out", "seed/judgements.jsonl", "where to write the fixtures")
	fs.Parse(args[1:])
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	prov, err := provider.New(provider.Config{Kind: *kind, Model: *model, BaseURL: *base})
	if err != nil {
		return err
	}
	d := &workflows.Deps{Store: e.store, Config: e.cfg, Provider: prov, Log: log}
	in, _, err := d.Input(ctx)
	if err != nil {
		return err
	}
	in.Judgements = map[string]resolver.Judgement{}
	m := resolver.Build(in)
	var js []resolver.Judgement
	for _, c := range m.Candidates {
		j, err := prov.Judge(ctx, c)
		if err != nil {
			return fmt.Errorf("%s: %w", c.Key, err)
		}
		fmt.Printf("  %-55s -> %-38s %.2f\n", c.Key, j.Match, j.Confidence)
		if j.Match != "" {
			js = append(js, j)
		}
	}
	if err := provider.Record(e.cfg.Path(*out), js); err != nil {
		return err
	}
	fmt.Printf("%d judgements with a match written to %s by %s/%s\n", len(js), *out, prov.Name(), prov.Model())
	return nil
}

func snapshotCmd(ctx context.Context, log *slog.Logger, args []string) error {
	e, err := open(ctx, log)
	if err != nil {
		return err
	}
	defer e.store.Close()
	if len(args) == 0 || args[0] == "list" {
		rows, err := e.store.Pool.Query(ctx, `select id, created_at, entities, edges, path from snapshot order by created_at desc`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, path string
			var at time.Time
			var ne, ed int
			rows.Scan(&id, &at, &ne, &ed, &path)
			fmt.Printf("%s  %s  %d entities  %d edges  %s\n", id, at.Format(time.DateTime), ne, ed, path)
		}
		return nil
	}
	if args[0] == "diff" && len(args) == 3 {
		return diffSnapshots(ctx, e, args[1], args[2])
	}
	return fmt.Errorf("usage: itx snapshot list | diff A B")
}
