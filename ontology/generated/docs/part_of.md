---
search:
  boost: 5.0
---

# Slot: part_of 


_The team belongs to this market unit._



<div data-search-exclude markdown="1">



URI: [ctx:part_of](https://example.org/it-intelligence/part_of)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Team](Team.md) | An accountable team |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [MarketUnit](MarketUnit.md) |
| Domain | [Team](Team.md) |
| Domain Of | [Team](Team.md) |










## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Team |
| ctx_range | MarketUnit |
| ctx_cardinality | one |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:part_of |
| native | ctx:part_of |




## LinkML Source

<details>
```yaml
name: part_of
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Team
  ctx_range:
    tag: ctx_range
    value: MarketUnit
  ctx_cardinality:
    tag: ctx_cardinality
    value: one
description: The team belongs to this market unit.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Team
domain_of:
- Team
range: MarketUnit

```
</details></div>