---
search:
  boost: 5.0
---

# Slot: built_from 


_The workload's image is built from this repository._



<div data-search-exclude markdown="1">



URI: [ctx:built_from](https://example.org/it-intelligence/built_from)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Workload](Workload.md) | A deployed unit on a cluster, such as a deployment, a stateful set or a cron ... |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Repository](Repository.md) |
| Domain | [Workload](Workload.md) |
| Domain Of | [Workload](Workload.md) |










## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Workload |
| ctx_range | Repository |
| ctx_cardinality | one |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:built_from |
| native | ctx:built_from |




## LinkML Source

<details>
```yaml
name: built_from
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Workload
  ctx_range:
    tag: ctx_range
    value: Repository
  ctx_cardinality:
    tag: ctx_cardinality
    value: one
description: The workload's image is built from this repository.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Workload
domain_of:
- Workload
range: Repository

```
</details></div>