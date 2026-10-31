---
search:
  boost: 5.0
---

# Slot: depends_on 


_The application calls, or needs, this other application._



<div data-search-exclude markdown="1">



URI: [ctx:depends_on](https://example.org/it-intelligence/depends_on)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Application](Application.md) |
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
| ctx_range | Application |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:depends_on |
| native | ctx:depends_on |




## LinkML Source

<details>
```yaml
name: depends_on
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application
  ctx_range:
    tag: ctx_range
    value: Application
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The application calls, or needs, this other application.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Application
domain_of:
- Application
range: Application
multivalued: true

```
</details></div>