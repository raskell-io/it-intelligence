---
search:
  boost: 5.0
---

# Slot: runs_in 


_The workload runs in this namespace._



<div data-search-exclude markdown="1">



URI: [ctx:runs_in](https://example.org/it-intelligence/runs_in)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Workload](Workload.md) | A deployed unit on a cluster, such as a deployment, a stateful set or a cron ... |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Namespace](Namespace.md) |
| Domain | [Workload](Workload.md) |
| Domain Of | [Workload](Workload.md) |










## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Workload |
| ctx_range | Namespace |
| ctx_cardinality | one |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:runs_in |
| native | ctx:runs_in |




## LinkML Source

<details>
```yaml
name: runs_in
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Workload
  ctx_range:
    tag: ctx_range
    value: Namespace
  ctx_cardinality:
    tag: ctx_cardinality
    value: one
description: The workload runs in this namespace.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Workload
domain_of:
- Workload
range: Namespace

```
</details></div>