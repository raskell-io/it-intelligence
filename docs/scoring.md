# Scoring sources

Not every system of record deserves the same trust. An inventory read from a cluster is close to ground
truth; an application list maintained by hand once a year is not. The score of each data set is
**computed, not declared**, on every ingest, and it travels into the model.

## The four signals

| Signal | Share of records that... | Computed from |
|---|---|---|
| **freshness** | changed within the data set's expected cycle | the field the mapper names as "modified" (`last_modified`, `pushed_at`, ...), else the observation time; the cycle is in `config/sources.yaml` |
| **completeness** | have every field the mapper needs | the mapper's `Fields(dataset).Required` |
| **agreement** | ... of the data set's relationships that another source also asserts | active edges from this data set, and whether another source has a row for the same relationship |
| **resolvability** | resolved: their identity and every reference landed on an entity | the build's `record_status` |

Agreement is left out for a data set that asserts no relationships, and the weights of the rest are
renormalised. **Overall** is the weighted mean; the weights are in `config/sources.yaml` (freshness 0.3,
completeness 0.2, agreement 0.25, resolvability 0.25).

Every number can be checked: each scorecard lists the records that fail each signal, and the scorecards
page shows them.

## On the Kestrel seed

```text
 data set                  fresh  complete  agree  resolve  overall
 earepo/capabilities        0.00    1.00    0.00    1.00     0.45
 earepo/teams               0.00    1.00    0.00    1.00     0.45
 earepo/interfaces          0.25    1.00    0.00    1.00     0.53
 earepo/applications        0.19    0.86    0.41    1.00     0.58
 earepo/market_units        0.00    1.00     n/a    1.00     0.60
 cluster/namespaces         1.00    1.00    0.00    1.00     0.75
 githost/repositories       0.96    1.00    0.39    1.00     0.84
 itsm/business_services     1.00    0.97    1.00    1.00     0.99
```

The EA repository scores worst, as it should: most of its records were last touched more than 180 days
ago. Note what agreement does to the cluster: its facts are true, but no other source can see a
namespace's owner label, so nobody confirms them. Agreement measures corroboration, not truth; that is
why it is one signal of four.

## How scores travel

The next build starts every asserted fact at `0.7 + 0.3 x overall` of the data set it came from. A fact
from a well-kept source therefore outranks the same claim from a neglected one, and when two sources
claim the same ownership, answers show the stronger. Derived facts inherit the weakest confidence in their
chain, so a neglected source weakens everything built on it.

## Why public

The scorecards are a page everyone can see. A data set that is accurate, curated and connected is used
by every architect, engineer and agent, scores well, and may end up declared primary. One that is stale
scores badly, and everyone can see that too. It turns data quality from an audit finding into something
closer to a league table.
