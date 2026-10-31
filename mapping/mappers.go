package mapping

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/connectors"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
)

// The six mappers of the reference. Each is short on purpose: read the payload, assert entities and
// edges, nothing else. Identity decisions beyond shared keys belong to the resolver.

const eaApps = "earepo:applications"

func init() {
	register(earepoMapper{})
	register(itsmMapper{})
	register(githostMapper{})
	register(clusterMapper{})
	register(gatewayMapper{})
	register(wafMapper{})
}

func decode(o connectors.Observation, v any) error {
	if err := json.Unmarshal(o.Payload, v); err != nil {
		return fmt.Errorf("%s/%s %s: %w", o.Source, o.Dataset, o.ExternalID, err)
	}
	return nil
}

func start(o connectors.Observation) Assertions { return Assertions{Observation: o} }

// ---- EA repository ----

type earepoMapper struct{}

func (earepoMapper) Source() string { return "earepo" }

func (earepoMapper) Fields(ds string) DatasetFields {
	switch ds {
	case "applications":
		return DatasetFields{Required: []string{"app_id", "name", "short_name", "lifecycle", "owner", "capabilities"}, Modified: "last_modified"}
	case "capabilities":
		return DatasetFields{Required: []string{"cap_id", "name", "level"}, Modified: "last_modified"}
	case "teams":
		return DatasetFields{Required: []string{"team_id", "name", "market_unit"}, Modified: "last_modified"}
	case "market_units":
		return DatasetFields{Required: []string{"mu_id", "name"}, Modified: "last_modified"}
	case "interfaces":
		return DatasetFields{Required: []string{"from_app", "to_app"}, Modified: "last_modified"}
	}
	return DatasetFields{}
}

func (earepoMapper) Map(ctx Context, o connectors.Observation) (Assertions, error) {
	a := start(o)
	var r map[string]string
	if err := decode(o, &r); err != nil {
		return a, err
	}
	switch o.Dataset {
	case "market_units":
		mu := a.Entity(vocab.MarketUnit, Slug(r["name"]), r["name"], nil)
		_ = mu
	case "teams":
		t := a.Entity(vocab.Team, Slug(r["name"]), r["name"], nil)
		if r["market_unit"] != "" {
			a.Edge(vocab.PartOf, t, Alias(vocab.MarketUnit, "earepo:market_units", r["market_unit"]), Asserted)
		}
	case "capabilities":
		level := 1
		fmt.Sscan(r["level"], &level)
		c := a.Entity(vocab.Capability, Slug(r["name"]), r["name"], map[string]any{"level": level})
		if r["parent_id"] != "" {
			a.Edge(vocab.Refines, c, Alias(vocab.Capability, "earepo:capabilities", r["parent_id"]), Asserted)
		}
	case "applications":
		app := a.Entity(vocab.Application, r["short_name"], r["name"], map[string]any{
			"lifecycle": r["lifecycle"], "description": r["description"], "ea_id": r["app_id"],
		})
		if r["owner"] != "" {
			a.Edge(vocab.OwnedBy, app, Key(vocab.Team, Slug(r["owner"])), Asserted)
		}
		for _, mu := range split(r["market_units"]) {
			a.Edge(vocab.OperatedBy, app, Alias(vocab.MarketUnit, "earepo:market_units", mu), Asserted)
		}
		for _, c := range split(r["capabilities"]) {
			a.Edge(vocab.Supports, app, Alias(vocab.Capability, "earepo:capabilities", c), Asserted)
		}
		for _, repo := range split(r["repositories"]) {
			a.Edge(vocab.RealisedBy, app, Key(vocab.Repository, repo), Asserted)
		}
		if r["successor"] != "" {
			a.Edge(vocab.ReplacedBy, app, Alias(vocab.Application, eaApps, r["successor"]), Asserted)
		}
	case "interfaces":
		a.Edge(vocab.DependsOn, Alias(vocab.Application, eaApps, r["from_app"]), Alias(vocab.Application, eaApps, r["to_app"]), Asserted)
	}
	return a, nil
}

// ---- IT service management ----

type itsmMapper struct{}

func (itsmMapper) Source() string { return "itsm" }

func (itsmMapper) Fields(ds string) DatasetFields {
	switch ds {
	case "business_services":
		return DatasetFields{Required: []string{"sys_id", "name", "support_group", "ea_reference"}, Modified: "sys_updated_on"}
	case "incidents":
		return DatasetFields{Required: []string{"number", "business_service", "opened_at"}, Modified: "opened_at"}
	}
	return DatasetFields{}
}

func (itsmMapper) Map(ctx Context, o connectors.Observation) (Assertions, error) {
	a := start(o)
	var r map[string]string
	if err := decode(o, &r); err != nil {
		return a, err
	}
	switch o.Dataset {
	case "business_services":
		var target Ref
		if r["ea_reference"] != "" {
			target = Alias(vocab.Application, eaApps, r["ea_reference"])
		}
		team := supportTeam(ctx.Config, r["support_group"])
		self := a.SameAs(vocab.Application, target, r["name"], map[string]string{
			"name": r["name"], "description": r["description"], "support_group": r["support_group"], "owner_team": team,
		})
		if team != "" {
			a.Edge(vocab.OwnedBy, self, Key(vocab.Team, team), Asserted)
		}
	case "incidents":
		opened, err := time.Parse(time.RFC3339, r["opened_at"])
		if err == nil && ctx.Now.Sub(opened) <= 90*24*time.Hour {
			a.Count(Alias(vocab.Application, "itsm:business_services", r["business_service"]), "incidents_90d", 1)
		}
	}
	return a, nil
}

func supportTeam(c Config, group string) string {
	for _, suf := range c.SupportGroupSuffixes {
		group = strings.TrimSuffix(group, suf)
	}
	return Slug(group)
}

// ---- git host ----

type githostMapper struct{}

func (githostMapper) Source() string { return "githost" }

func (githostMapper) Fields(ds string) DatasetFields {
	switch ds {
	case "repositories":
		return DatasetFields{Required: []string{"full_name", "team", "description", "pushed_at"}, Modified: "pushed_at"}
	case "teams":
		return DatasetFields{Required: []string{"slug", "name"}}
	case "decisions":
		return DatasetFields{Required: []string{"repository", "number", "title", "status", "constrains"}, Modified: "updated_at"}
	}
	return DatasetFields{}
}

func (githostMapper) Map(ctx Context, o connectors.Observation) (Assertions, error) {
	a := start(o)
	switch o.Dataset {
	case "teams":
		var t struct{ Slug, Name string }
		if err := decode(o, &t); err != nil {
			return a, err
		}
		a.Entity(vocab.Team, t.Slug, t.Name, nil)
	case "repositories":
		var r struct {
			FullName      string            `json:"full_name"`
			Name          string            `json:"name"`
			Team          string            `json:"team"`
			Description   string            `json:"description"`
			Readme        string            `json:"readme_excerpt"`
			Archived      bool              `json:"archived"`
			PushedAt      string            `json:"pushed_at"`
			Committers12m *int              `json:"committers_12m"`
			Catalog       map[string]string `json:"catalog_info"`
		}
		if err := decode(o, &r); err != nil {
			return a, err
		}
		attrs := map[string]any{"description": r.Description, "last_commit": r.PushedAt, "archived": r.Archived}
		if r.Committers12m != nil {
			attrs["committers_12m"] = *r.Committers12m
			// The truck factor needs per-author commit shares; the seed only has distinct committers, which
			// is an upper bound and good enough to find the repositories that hang on one person.
			attrs["truck_factor"] = *r.Committers12m
		}
		repo := a.Entity(vocab.Repository, r.FullName, r.Name, attrs)
		a.Entities[0].Hints = map[string]string{"name": r.Name, "readme": r.Readme, "owner_team": r.Team}
		if r.Team != "" {
			a.Edge(vocab.OwnedBy, repo, Key(vocab.Team, r.Team), Asserted)
		}
		if id := r.Catalog["ea_application_id"]; id != "" {
			a.Edge(vocab.RealisedBy, Alias(vocab.Application, eaApps, id), repo, Asserted)
		}
	case "decisions":
		var d struct {
			Repository string   `json:"repository"`
			Number     int      `json:"number"`
			Title      string   `json:"title"`
			Status     string   `json:"status"`
			Date       string   `json:"date"`
			Constrains []string `json:"constrains"`
			Supersedes int      `json:"supersedes"`
			URL        string   `json:"url"`
		}
		if err := decode(o, &d); err != nil {
			return a, err
		}
		dec := a.Entity(vocab.Decision, fmt.Sprintf("%s/%d", d.Repository, d.Number), d.Title,
			map[string]any{"status": d.Status, "decided_on": d.Date, "url": d.URL})
		for _, id := range d.Constrains {
			a.Edge(vocab.Constrains, dec, Alias(vocab.Application, eaApps, id), Asserted)
		}
		if d.Supersedes != 0 {
			a.Edge(vocab.Supersedes, dec, Key(vocab.Decision, fmt.Sprintf("%s/%d", d.Repository, d.Supersedes)), Asserted)
		}
	}
	return a, nil
}

// ---- container clusters ----

type clusterMapper struct{}

func (clusterMapper) Source() string { return "cluster" }

func (clusterMapper) Fields(ds string) DatasetFields {
	switch ds {
	case "namespaces":
		return DatasetFields{Required: []string{"metadata.name", "metadata.labels"}}
	case "workloads":
		return DatasetFields{Required: []string{"metadata.name", "metadata.namespace", "metadata.labels"}}
	}
	return DatasetFields{}
}

type k8sObject struct {
	Kind     string `json:"kind"`
	Cluster  string `json:"cluster"`
	Metadata struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Labels    map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Replicas *int   `json:"replicas"`
		Image    string `json:"image"`
	} `json:"spec"`
}

func (clusterMapper) Map(ctx Context, o connectors.Observation) (Assertions, error) {
	a := start(o)
	var k k8sObject
	if err := decode(o, &k); err != nil {
		return a, err
	}
	cfg := ctx.Config
	switch o.Dataset {
	case "namespaces":
		ns := a.Entity(vocab.Namespace, k.Metadata.Name, k.Metadata.Name, map[string]any{"cluster": k.Cluster})
		if owner := k.Metadata.Labels[cfg.NamespaceOwnerLabel]; owner != "" {
			a.Edge(vocab.OwnedBy, ns, Key(vocab.Team, owner), Asserted)
		}
	case "workloads":
		attrs := map[string]any{"workload_type": k.Kind, "image": k.Spec.Image}
		if k.Spec.Replicas != nil {
			attrs["replicas"] = *k.Spec.Replicas
		}
		wl := a.Entity(vocab.Workload, k.Metadata.Namespace+"/"+k.Metadata.Name, k.Metadata.Name, attrs)
		a.Entities[0].Hints = map[string]string{"name": k.Metadata.Name, "namespace": k.Metadata.Namespace, "image": k.Spec.Image}
		a.Edge(vocab.RunsIn, wl, Key(vocab.Namespace, k.Metadata.Namespace), Asserted)
		// The cheapest win in the whole system: a label carrying the EA application id turns an
		// inference into an assertion.
		if app := k.Metadata.Labels[cfg.ApplicationLabel]; app != "" {
			if cfg.IsEAID(app) {
				a.Edge(vocab.DeployedAs, Alias(vocab.Application, eaApps, app), wl, Asserted)
			} else {
				// An application id the EA repository does not use: an application nobody registered.
				ref := Key(vocab.Application, Slug(app))
				a.Entities = append(a.Entities, EntityAssertion{Ref: ref, Name: app, Attributes: map[string]any{"not_in_ea": true}})
				a.Edge(vocab.DeployedAs, ref, wl, Asserted)
			}
		}
		if repo := k.Metadata.Labels[cfg.SourceRepositoryLabel]; repo != "" {
			a.Edge(vocab.BuiltFrom, wl, Key(vocab.Repository, repo), Asserted)
		}
	}
	return a, nil
}

// ---- API gateway ----

type gatewayMapper struct{}

func (gatewayMapper) Source() string { return "gateway" }

func (gatewayMapper) Fields(string) DatasetFields {
	return DatasetFields{Required: []string{"host", "path", "upstream.namespace", "upstream.service", "visibility"}, Modified: "updated_at"}
}

func (gatewayMapper) Map(ctx Context, o connectors.Observation) (Assertions, error) {
	a := start(o)
	var r struct {
		Host, Path, Visibility string
		Upstream               struct{ Namespace, Service string }
	}
	if err := decode(o, &r); err != nil {
		return a, err
	}
	exposure := "internal"
	if r.Visibility == "public" {
		exposure = "internet"
	}
	ep := a.Entity(vocab.Endpoint, r.Host+r.Path, r.Host+r.Path, map[string]any{"exposure": exposure})
	if r.Upstream.Namespace != "" && r.Upstream.Service != "" {
		a.Edge(vocab.RoutesTo, ep, Key(vocab.Workload, r.Upstream.Namespace+"/"+r.Upstream.Service), Asserted)
	}
	return a, nil
}

// ---- web application firewall ----

type wafMapper struct{}

func (wafMapper) Source() string { return "waf" }

func (wafMapper) Fields(string) DatasetFields {
	return DatasetFields{Required: []string{"policy_id", "host", "path", "mode"}, Modified: "updated_at"}
}

func (wafMapper) Map(ctx Context, o connectors.Observation) (Assertions, error) {
	a := start(o)
	var p struct{ Host, Path, Mode string }
	if err := decode(o, &p); err != nil {
		return a, err
	}
	a.Entity(vocab.Endpoint, p.Host+p.Path, p.Host+p.Path, map[string]any{"waf": p.Mode})
	return a, nil
}
