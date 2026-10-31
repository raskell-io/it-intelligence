package mapping

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/raskell-io/it-intelligence/connectors"
)

func obs(source, dataset, id string, payload any) connectors.Observation {
	b, _ := json.Marshal(payload)
	return connectors.Observation{ID: 1, Source: source, Dataset: dataset, ExternalID: id, Payload: b, ObservedAt: time.Now()}
}

func hasEdge(a Assertions, kind, from, to string) bool {
	for _, e := range a.Edges {
		if e.Kind == kind && refString(e.From) == from && refString(e.To) == to {
			return true
		}
	}
	return false
}

func refString(r Ref) string {
	if r.Key != "" {
		return r.ID()
	}
	return r.Alias()
}

func TestMappers(t *testing.T) {
	ctx := Context{Now: time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC), Config: DefaultConfig()}
	tests := []struct {
		name  string
		obs   connectors.Observation
		check func(t *testing.T, a Assertions)
	}{
		{"a deployment label with an EA id is an asserted deployed_as",
			obs("cluster", "workloads", "claims-prod/policy-admin", map[string]any{"kind": "Deployment", "metadata": map[string]any{
				"name": "policy-admin", "namespace": "claims-prod",
				"labels": map[string]string{"app.kestrel.example/application-id": "APP-0142", "source.kestrel.example/repository": "kestrel/pol-adm-svc"}}}),
			func(t *testing.T, a Assertions) {
				if a.Entities[0].Ref.ID() != "ctx:Workload/claims-prod/policy-admin" || !a.Entities[0].Primary {
					t.Errorf("primary entity = %+v", a.Entities[0])
				}
				if !hasEdge(a, "deployed_as", "earepo:applications/APP-0142", "ctx:Workload/claims-prod/policy-admin") {
					t.Error("missing deployed_as by EA id")
				}
				if !hasEdge(a, "runs_in", "ctx:Workload/claims-prod/policy-admin", "ctx:Namespace/claims-prod") {
					t.Error("missing runs_in")
				}
				if !hasEdge(a, "built_from", "ctx:Workload/claims-prod/policy-admin", "ctx:Repository/kestrel/pol-adm-svc") {
					t.Error("missing built_from")
				}
			}},
		{"an application id the EA repository does not use creates an application not in EA",
			obs("cluster", "workloads", "south-int/camel-hub", map[string]any{"kind": "Deployment", "metadata": map[string]any{
				"name": "camel-hub", "namespace": "south-int", "labels": map[string]string{"app.kestrel.example/application-id": "south-integration-hub"}}}),
			func(t *testing.T, a Assertions) {
				if len(a.Entities) != 2 || a.Entities[1].Ref.ID() != "ctx:Application/south-integration-hub" || a.Entities[1].Attributes["not_in_ea"] != true {
					t.Errorf("entities = %+v", a.Entities)
				}
			}},
		{"a business service without an EA reference is an identity candidate",
			obs("itsm", "business_services", "BS-1042", map[string]string{"sys_id": "BS-1042", "name": "PAS", "support_group": "Claims Platform Support"}),
			func(t *testing.T, a Assertions) {
				if !a.Entities[0].Ref.IsZero() || a.Entities[0].Ref.Kind != "Application" {
					t.Errorf("want a zero ref, got %+v", a.Entities[0].Ref)
				}
				if !hasEdge(a, "owned_by", "itsm:business_services/BS-1042", "ctx:Team/claims-platform") {
					t.Errorf("support group should map to the team: %+v", a.Edges)
				}
			}},
		{"an EA application asserts owner, market units, capabilities and repositories",
			obs("earepo", "applications", "APP-0142", map[string]string{"app_id": "APP-0142", "name": "Policy Administration", "short_name": "policy-admin",
				"lifecycle": "production", "owner": "Claims Platform", "market_units": "MU-C;MU-N", "capabilities": "CAP-01.2", "repositories": "kestrel/pol-adm-svc"}),
			func(t *testing.T, a Assertions) {
				if a.Entities[0].Ref.ID() != "ctx:Application/policy-admin" || a.Entities[0].Attributes["lifecycle"] != "production" {
					t.Errorf("entity = %+v", a.Entities[0])
				}
				for _, want := range [][3]string{
					{"owned_by", "ctx:Application/policy-admin", "ctx:Team/claims-platform"},
					{"operated_by", "ctx:Application/policy-admin", "earepo:market_units/MU-N"},
					{"supports", "ctx:Application/policy-admin", "earepo:capabilities/CAP-01.2"},
					{"realised_by", "ctx:Application/policy-admin", "ctx:Repository/kestrel/pol-adm-svc"},
				} {
					if !hasEdge(a, want[0], want[1], want[2]) {
						t.Errorf("missing %v", want)
					}
				}
			}},
		{"incidents older than 90 days do not count",
			obs("itsm", "incidents", "INC1", map[string]string{"number": "INC1", "business_service": "BS-1", "opened_at": "2026-01-01T00:00:00Z"}),
			func(t *testing.T, a Assertions) {
				if len(a.Counters) != 0 {
					t.Errorf("counters = %+v", a.Counters)
				}
			}},
		{"a WAF policy sets the mode of its endpoint",
			obs("waf", "policies", "waf-1", map[string]string{"policy_id": "waf-1", "host": "api.kestrel.example", "path": "/policies", "mode": "enforced"}),
			func(t *testing.T, a Assertions) {
				if a.Entities[0].Ref.ID() != "ctx:Endpoint/api.kestrel.example/policies" || a.Entities[0].Attributes["waf"] != "enforced" {
					t.Errorf("entity = %+v", a.Entities[0])
				}
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := For(tc.obs.Source)
			if !ok {
				t.Fatalf("no mapper for %s", tc.obs.Source)
			}
			a, err := m.Map(ctx, tc.obs)
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, a)
		})
	}
}
