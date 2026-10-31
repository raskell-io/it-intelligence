---
search:
  boost: 5.0
---

# Slot: exposed_via 


_The application is reachable through this endpoint._



<div data-search-exclude markdown="1">



URI: [ctx:exposed_via](https://example.org/it-intelligence/exposed_via)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Endpoint](Endpoint.md) |
| Domain | [Application](Application.md) |
| Domain Of | [Application](Application.md) |
### Cardinality and Requirements

| Property | Value |
| --- | --- |
| Multivalued | Yes |












## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Application |
| ctx_range | Endpoint |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:exposed_via |
| native | ctx:exposed_via |




## LinkML Source

<details>
```yaml
name: exposed_via
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application
  ctx_range:
    tag: ctx_range
    value: Endpoint
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The application is reachable through this endpoint.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Application
domain_of:
- Application
range: Endpoint
multivalued: true

```
</details></div>