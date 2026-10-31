---
search:
  boost: 5.0
---

# Slot: id 


_The Context Model id, for example ctx:Application/policy-admin._



<div data-search-exclude markdown="1">



URI: [ctx:id](https://example.org/it-intelligence/id)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Entity](Entity.md) | Something the Context Model concluded exists, from one or more observations |  no  |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |
| [Repository](Repository.md) | A code repository on the git host |  no  |
| [Workload](Workload.md) | A deployed unit on a cluster, such as a deployment, a stateful set or a cron ... |  no  |
| [Namespace](Namespace.md) | A namespace on a container cluster |  no  |
| [Team](Team.md) | An accountable team |  no  |
| [Capability](Capability.md) | A business capability from the enterprise architecture capability map |  no  |
| [Endpoint](Endpoint.md) | A reachable host and path prefix |  no  |
| [Decision](Decision.md) | An architecture decision record, with the entities it constrains |  no  |
| [MarketUnit](MarketUnit.md) | An organisational unit of the group, usually a country or a region |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [String](String.md) |
| Domain Of | [Entity](Entity.md) |
### Cardinality and Requirements

| Property | Value |
| --- | --- |
| Required | Yes |


### Slot Characteristics

| Property | Value |
| --- | --- |
| Identifier | Yes |












## Identifier and Mapping Information





### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:id |
| native | ctx:id |




## LinkML Source

<details>
```yaml
name: id
description: The Context Model id, for example ctx:Application/policy-admin.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
identifier: true
domain_of:
- Entity
range: string
required: true

```
</details></div>