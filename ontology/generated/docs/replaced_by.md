---
search:
  boost: 5.0
---

# Slot: replaced_by 


_The application is, or will be, replaced by this successor._



<div data-search-exclude markdown="1">



URI: [ctx:replaced_by](https://example.org/it-intelligence/replaced_by)
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










## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_domain | Application |
| ctx_range | Application |
| ctx_cardinality | one |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:replaced_by |
| native | ctx:replaced_by |




## LinkML Source

<details>
```yaml
name: replaced_by
annotations:
  ctx_domain:
    tag: ctx_domain
    value: Application
  ctx_range:
    tag: ctx_range
    value: Application
  ctx_cardinality:
    tag: ctx_cardinality
    value: one
description: The application is, or will be, replaced by this successor.
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
domain: Application
domain_of:
- Application
range: Application

```
</details></div>