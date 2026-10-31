---
search:
  boost: 5.0
---

# Slot: routes_to 


_The endpoint sends traffic to this workload._



<div data-search-exclude markdown="1">



URI: [ctx:routes_to](https://example.org/it-intelligence/routes_to)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Endpoint](Endpoint.md) | A reachable host and path prefix |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Workload](Workload.md) |
| Domain | [Endpoint](Endpoint.md) |
| Domain Of | [Endpoint](Endpoint.md) |
### Cardinality and Requirements

| Property | Value |
| --- | --- |
| Multivalued | Yes |












## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Endpoint |
| ctx_range | Workload |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:routes_to |
| native | ctx:routes_to |




## LinkML Source

<details>
```yaml
name: routes_to
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Endpoint
  ctx_range:
    tag: ctx_range
    value: Workload
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The endpoint sends traffic to this workload.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Endpoint
domain_of:
- Endpoint
range: Workload
multivalued: true

```
</details></div>