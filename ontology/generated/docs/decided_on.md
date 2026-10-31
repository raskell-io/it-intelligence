---
search:
  boost: 5.0
---

# Slot: decided_on 

<div data-search-exclude markdown="1">



URI: [ctx:decided_on](https://example.org/it-intelligence/decided_on)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Decision](Decision.md) | An architecture decision record, with the entities it constrains |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Date](Date.md) |
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
| self | ctx:decided_on |
| native | ctx:decided_on |




## LinkML Source

<details>
```yaml
name: decided_on
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
owner: Decision
domain_of:
- Decision
range: date

```
</details></div>