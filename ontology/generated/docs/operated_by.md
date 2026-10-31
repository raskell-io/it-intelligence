---
search:
  boost: 5.0
---

# Slot: operated_by 


_The application is operated by, and serves, this market unit._



<div data-search-exclude markdown="1">



URI: [ctx:operated_by](https://example.org/it-intelligence/operated_by)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Application](Application.md) | A thing the business recognises as one system, however many parts it has |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [MarketUnit](MarketUnit.md) |
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
| ctx_range | MarketUnit |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:operated_by |
| native | ctx:operated_by |




## LinkML Source

<details>
```yaml
name: operated_by
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application
  ctx_range:
    tag: ctx_range
    value: MarketUnit
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The application is operated by, and serves, this market unit.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Application
domain_of:
- Application
range: MarketUnit
multivalued: true

```
</details></div>