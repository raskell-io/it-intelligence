---
search:
  boost: 5.0
---

# Slot: supersedes 


_The decision replaces this earlier decision._



<div data-search-exclude markdown="1">



URI: [ctx:supersedes](https://example.org/it-intelligence/supersedes)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Decision](Decision.md) | An architecture decision record, with the entities it constrains |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Decision](Decision.md) |
| Domain | [Decision](Decision.md) |
| Domain Of | [Decision](Decision.md) |










## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Decision |
| ctx_range | Decision |
| ctx_cardinality | one |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:supersedes |
| native | ctx:supersedes |




## LinkML Source

<details>
```yaml
name: supersedes
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Decision
  ctx_range:
    tag: ctx_range
    value: Decision
  ctx_cardinality:
    tag: ctx_cardinality
    value: one
description: The decision replaces this earlier decision.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Decision
domain_of:
- Decision
range: Decision

```
</details></div>