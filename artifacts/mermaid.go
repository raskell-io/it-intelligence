package artifacts

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/raskell-io/it-intelligence/api/query"
)

var nonIdent = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func nodeID(id string) string {
	_, rest, _ := strings.Cut(id, ":")
	return strings.Trim(strings.ToLower(nonIdent.ReplaceAllString(rest, "_")), "_")
}

func label(s string) string { return strings.ReplaceAll(s, `"`, "'") }

// Mermaid draws an entity's neighbourhood as a Mermaid flowchart. Inferred edges are dashed and carry
// their confidence, so the diagram is honest about what it knows. Because it is text, it renders in the
// git host, the wiki and the chat tool, and two snapshots can be diffed.
func Mermaid(ctx context.Context, q *query.Q, id string) (string, error) {
	e, err := q.GetEntity(ctx, id)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	self := nodeID(e.ID)
	title := e.Name
	if o, err := q.FindOwner(ctx, e.ID); err == nil && o.Owner != "" {
		title += "<br/>owner: " + strings.TrimPrefix(o.Owner, "ctx:Team/")
	}
	fmt.Fprintf(&b, "  %s[\"%s\"]\n", self, label(title))

	order := []string{"realised_by", "deployed_as", "exposed_via", "supports", "depends_on", "replaced_by", "constrains"}
	var nodes, links []string
	seen := map[string]bool{self: true}
	for _, kind := range order {
		for _, r := range e.Edges {
			if r.Kind != kind || r.Status == "rejected" {
				continue
			}
			if kind != "constrains" && r.Direction != "out" {
				continue
			}
			if kind == "constrains" && r.Direction != "in" {
				continue
			}
			n := nodeID(r.Other)
			if !seen[n] {
				seen[n] = true
				nodes = append(nodes, fmt.Sprintf("  %s[\"%s\"]", n, label(nodeLabel(ctx, q, r))))
			}
			arrow, text := "-->", kind
			if r.Basis == "inferred" {
				arrow = "-.->"
				text = fmt.Sprintf("%s, inferred %.2f", kind, r.Confidence)
				if r.Status == "pending" {
					text += ", pending"
				}
			}
			if kind == "constrains" {
				links = append(links, fmt.Sprintf("  %s %s|%s| %s", n, arrow, text, self))
			} else if !contains(links, fmt.Sprintf("  %s %s|%s| %s", self, arrow, text, n)) {
				links = append(links, fmt.Sprintf("  %s %s|%s| %s", self, arrow, text, n))
			}
		}
	}
	for _, n := range nodes {
		b.WriteString(n + "\n")
	}
	if len(links) > 0 {
		b.WriteString("\n")
	}
	for _, l := range links {
		b.WriteString(l + "\n")
	}
	return b.String(), nil
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func nodeLabel(ctx context.Context, q *query.Q, r query.EdgeRef) string {
	kind, rest, _ := strings.Cut(strings.TrimPrefix(r.Other, "ctx:"), "/")
	switch kind {
	case "Repository":
		return "repo: " + r.OtherName
	case "Workload":
		ns, _, _ := strings.Cut(rest, "/")
		return "workload: " + r.OtherName + "<br/>ns: " + ns
	case "Endpoint":
		waf := "none"
		if o, err := q.GetEntity(ctx, r.Other); err == nil {
			if w, ok := o.Attributes["waf"].(string); ok {
				waf = w
			}
		}
		return "endpoint: " + r.OtherName + "<br/>WAF: " + waf
	case "Capability":
		return "capability: " + r.OtherName
	case "Decision":
		return "decision: " + r.OtherName
	}
	return r.OtherName
}
