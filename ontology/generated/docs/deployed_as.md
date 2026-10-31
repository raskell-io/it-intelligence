---
search:
  boost: 5.0
---

# Slot: deployed_as 


_The application runs as this workload._



<div data-search-exclude markdown="1">



URI: [ctx:deployed_as](https://example.org/it-intelligence/deployed_as)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Workload](Workload.md) |
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
| ctx_range | Workload |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:deployed_as |
| native | ctx:deployed_as |




## LinkML Source

<details>
```yaml
name: deployed_as
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application
  ctx_range:
    tag: ctx_range
    value: Workload
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The application runs as this workload.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Application
domain_of:
- Application
range: Workload
multivalued: true

```
</details></div>