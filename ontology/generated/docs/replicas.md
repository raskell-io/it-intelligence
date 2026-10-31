---
search:
  boost: 5.0
---

# Slot: replicas 

<div data-search-exclude markdown="1">



URI: [ctx:replicas](https://example.org/it-intelligence/replicas)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Workload](Workload.md) | A deployed unit on a cluster, such as a deployment, a stateful set or a cron ... |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Integer](Integer.md) |
| Domain Of | [Workload](Workload.md) |
### Slot Characteristics

| Property | Value |
| --- | --- |
| Owner | [Workload](Workload.md) |












## Identifier and Mapping Information





### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:replicas |
| native | ctx:replicas |




## LinkML Source

<details>
```yaml
name: replicas
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
owner: Workload
domain_of:
- Workload
range: integer

```
</details></div>