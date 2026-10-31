package resolver

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
)

// PromptVersion goes into every inferred link's evidence. Change it whenever resolver/prompts changes.
const PromptVersion = "link-v1"

// inference turns what keys and rules could not settle into candidates, applies the stored judgement
// of each, and then the decisions people made about the result.
func (b *builder) inference(identity []mapping.Assertions) {
	ix := b.index()
	var cands []Candidate

	// Identity candidates: records that are some application, but nobody says which.
	for _, a := range identity {
		for _, ea := range a.Entities {
			if !ea.Ref.IsZero() {
				continue
			}
			c := Candidate{Key: a.OwnAlias(), Relation: SameAs, Name: ea.Name, Hints: ea.Hints, Team: ea.Hints["owner_team"]}
			cands = append(cands, c)
		}
	}
	// Repositories that realise no application, and workloads no application is deployed as.
	for _, r := range b.entitiesOf(vocab.Repository) {
		if ix.linked[vocab.RealisedBy+"|"+r.ID] {
			continue
		}
		team := ""
		if own := best(ix.Out(vocab.OwnedBy, r.ID)); own != nil {
			team = strings.TrimPrefix(own.To, "ctx:Team/")
		}
		cands = append(cands, Candidate{Key: "githost:repositories/" + strings.TrimPrefix(r.ID, "ctx:Repository/"),
			Relation: vocab.RealisedBy, Subject: r.ID, Name: r.Name, Hints: r.Hints, Team: team})
	}
	for _, w := range b.entitiesOf(vocab.Workload) {
		if ix.linked[vocab.DeployedAs+"|"+w.ID] {
			continue
		}
		team := ""
		if ri := best(ix.Out(vocab.RunsIn, w.ID)); ri != nil {
			if own := best(ix.Out(vocab.OwnedBy, ri.To)); own != nil {
				team = strings.TrimPrefix(own.To, "ctx:Team/")
			}
		}
		cands = append(cands, Candidate{Key: "cluster:workloads/" + strings.TrimPrefix(w.ID, "ctx:Workload/"),
			Relation: vocab.DeployedAs, Subject: w.ID, Name: w.Name, Hints: w.Hints, Team: team})
	}

	for i := range cands {
		c := &cands[i]
		c.Shortlist = b.shortlist(ix, c)
		var ids []string
		for _, m := range c.Shortlist {
			ids = append(ids, m.ID)
		}
		c.InputHash = hash(c.Key, c.Relation, c.Hints, ids)[:16]
		j, ok := b.in.Judgements[c.Key]
		if ok && j.InputHash == c.InputHash {
			c.Judged = true
			b.apply(c, j)
		}
	}
	b.m.Candidates = cands
}

// shortlist picks the applications a model should consider: those of the owning team first, then the
// most similar names. Small on purpose; the prompt stays cheap and the judgement easy to check.
func (b *builder) shortlist(ix *index, c *Candidate) []Match {
	type scored struct {
		e     *Entity
		score float64
	}
	name := c.Name
	if n := c.Hints["name"]; n != "" {
		name = n
	}
	var all []scored
	for _, app := range b.entitiesOf(vocab.Application) {
		s := similarity(b.normalise(name), appKey(app.ID))
		if t := similarity(strings.ToLower(name), strings.ToLower(app.Name)); t > s {
			s = t
		}
		if own := best(ix.Out(vocab.OwnedBy, app.ID)); own != nil && c.Team != "" && own.To == "ctx:Team/"+c.Team {
			s += 1
		}
		all = append(all, scored{app, s})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].score != all[j].score {
			return all[i].score > all[j].score
		}
		return all[i].e.ID < all[j].e.ID
	})
	n := b.in.Config.ShortlistSize
	var out []Match
	for _, s := range all {
		if len(out) == n || s.score <= 0.05 {
			break
		}
		hints := map[string]string{"description": fmt.Sprint(s.e.Attributes["description"])}
		if own := best(ix.Out(vocab.OwnedBy, s.e.ID)); own != nil {
			hints["owner_team"] = strings.TrimPrefix(own.To, "ctx:Team/")
		}
		out = append(out, Match{ID: s.e.ID, Name: s.e.Name, Hints: hints})
	}
	return out
}

// similarity is the Jaccard index of character trigrams.
func similarity(a, b string) float64 {
	ta, tb := trigrams(a), trigrams(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for t := range ta {
		if tb[t] {
			inter++
		}
	}
	return float64(inter) / float64(len(ta)+len(tb)-inter)
}

func trigrams(s string) map[string]bool {
	s = "  " + s + " "
	out := map[string]bool{}
	for i := 0; i+3 <= len(s); i++ {
		out[s[i:i+3]] = true
	}
	return out
}

// apply turns a judgement into an inferred link, respecting the thresholds and the decisions log.
func (b *builder) apply(c *Candidate, j Judgement) {
	if j.Match == "" {
		return
	}
	if !inShortlist(c, j.Match) {
		b.drop("judgement outside the shortlist", c.Key+" -> "+j.Match)
		return
	}
	if j.Confidence < b.in.Config.Review {
		b.drop("below the review threshold", fmt.Sprintf("%s -> %s at %.2f", c.Key, j.Match, j.Confidence))
		return
	}
	status := Pending
	if j.Confidence >= b.in.Config.Accept {
		status = Active
	}
	ev := map[string]any{"judgement": map[string]any{"match": j.Match, "confidence": j.Confidence, "reasons": j.Reasons},
		"provider": j.Provider, "model": j.Model, "prompt_version": j.PromptVersion, "input_hash": j.InputHash, "candidate": c.Key}

	if c.Relation == SameAs {
		src, rest, _ := strings.Cut(c.Key, ":")
		ds, ext, _ := strings.Cut(rest, "/")
		al := &Alias{EntityID: j.Match, Source: src, Dataset: ds, ExternalID: ext, Basis: mapping.Inferred,
			Confidence: j.Confidence, Status: status, Evidence: ev}
		al.ID = AliasID(j.Match, al.AliasString())
		b.decide("alias", al.ID, j.InputHash, &al.Status, &al.Basis, &al.Confidence, &al.ConfirmedBy, &al.ConfirmedAt, ev)
		b.m.Aliases[al.ID] = al
		if al.Status == Active {
			b.aliases[al.AliasString()] = j.Match
			b.m.Entities[j.Match].Sources[src] = true
			b.identified[c.Key] = j.Match
		}
		return
	}
	e := b.addEdge(c.Relation, j.Match, c.Subject, mapping.Inferred, j.Confidence, status, "resolver", ev, time.Time{})
	if e != nil {
		b.decide("edge", e.ID, j.InputHash, &e.Status, &e.Basis, &e.Confidence, &e.ConfirmedBy, &e.ConfirmedAt, ev)
	}
}

func inShortlist(c *Candidate, id string) bool {
	for _, m := range c.Shortlist {
		if m.ID == id {
			return true
		}
	}
	return false
}

// decide applies the latest human decision. A confirmation makes the link asserted, with a name on it.
// A rejection sticks as long as the evidence is the same; new evidence brings the link back to the
// queue, with the earlier rejection attached.
func (b *builder) decide(kind, id, evidenceHash string, status *string, basis *mapping.Basis, conf *float64,
	by *string, at **time.Time, ev map[string]any) {
	d, ok := b.decisions[id]
	if !ok || d.TargetType != kind {
		return
	}
	switch d.Action {
	case "confirm":
		*status, *basis, *conf, *by = Active, mapping.Asserted, 1.0, d.Actor
		t := d.At
		*at = &t
		ev["confirmed"] = map[string]any{"by": d.Actor, "at": d.At, "note": d.Note}
	case "reject":
		if d.EvidenceHash == evidenceHash {
			*status = Rejected
			ev["rejected"] = map[string]any{"by": d.Actor, "at": d.At, "note": d.Note}
		} else {
			*status = Pending
			ev["previously_rejected"] = map[string]any{"by": d.Actor, "at": d.At, "note": d.Note}
		}
	}
}
