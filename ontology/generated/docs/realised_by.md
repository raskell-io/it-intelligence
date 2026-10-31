---
search:
  boost: 5.0
---

# Slot: realised_by 


_The application is implemented in this repository._



<div data-search-exclude markdown="1">



URI: [ctx:realised_by](https://example.org/it-intelligence/realised_by)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Repository](Repository.md) |
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
| ctx_range | Repository |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:realised_by |
| native | ctx:realised_by |




## LinkML Source

<details>
```yaml
name: realised_by
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application
  ctx_range:
    tag: ctx_range
    value: Repository
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The application is implemented in this repository.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Application
domain_of:
- Application
range: Repository
multivalued: true

```
</details></div>