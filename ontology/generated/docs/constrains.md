---
search:
  boost: 5.0
---

# Slot: constrains 


_The decision applies to this entity._



<div data-search-exclude markdown="1">



URI: [ctx:constrains](https://example.org/it-intelligence/constrains)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Decision](Decision.md) | An architecture decision record, with the entities it constrains |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Entity](Entity.md) |
| Domain | [Decision](Decision.md) |
| Domain Of | [Decision](Decision.md) |
### Cardinality and Requirements

| Property | Value |
| --- | --- |
| Multivalued | Yes |












## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Decision |
| ctx_range | Entity |
| ctx_cardinality | many |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:constrains |
| native | ctx:constrains |




## LinkML Source

<details>
```yaml
name: constrains
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Decision
  ctx_range:
    tag: ctx_range
    value: Entity
  ctx_cardinality:
    tag: ctx_cardinality
    value: many
description: The decision applies to this entity.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Decision
domain_of:
- Decision
range: Entity
multivalued: true

```
</details></div>