---
search:
  boost: 5.0
---

# Slot: status 

<div data-search-exclude markdown="1">



URI: [ctx:status](https://example.org/it-intelligence/status)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Decision](Decision.md) | An architecture decision record, with the entities it constrains |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [DecisionStatus](DecisionStatus.md) |
| Domain Of | [Decision](Decision.md) |
### Slot Characteristics

| Property | Value |
| --- | --- |
| Owner | [Decision](Decision.md) |












## Identifier and Mapping Information





### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:status |
| native | ctx:status |




## LinkML Source

<details>
```yaml
name: status
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
owner: Decision
domain_of:
- Decision
range: DecisionStatus

```
</details></div>