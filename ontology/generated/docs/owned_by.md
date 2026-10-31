---
search:
  boost: 5.0
---

# Slot: owned_by 


_The accountable team._



<div data-search-exclude markdown="1">



URI: [ctx:owned_by](https://example.org/it-intelligence/owned_by)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |
| [Repository](Repository.md) | A code repository on the git host |  no  |
| [Workload](Workload.md) | A deployed unit on a cluster, such as a deployment, a stateful set or a cron ... |  no  |
| [Namespace](Namespace.md) | A namespace on a container cluster |  no  |
| [Endpoint](Endpoint.md) | A reachable host and path prefix |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Team](Team.md) |
| Domain Of | [Application](Application.md), [Repository](Repository.md), [Workload](Workload.md), [Namespace](Namespace.md), [Endpoint](Endpoint.md) |










## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Application Repository Workload Namespace Endpoint |
| ctx_range | Team |
| ctx_cardinality | one |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:owned_by |
| native | ctx:owned_by |




## LinkML Source

<details>
```yaml
name: owned_by
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application Repository Workload Namespace Endpoint
  ctx_range:
    tag: ctx_range
    value: Team
  ctx_cardinality:
    tag: ctx_cardinality
    value: one
description: The accountable team.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain_of:
- Application
- Repository
- Workload
- Namespace
- Endpoint
range: Team

```
</details></div>