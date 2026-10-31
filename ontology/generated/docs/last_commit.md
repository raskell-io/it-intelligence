---
search:
  boost: 5.0
---

# Slot: last_commit 

<div data-search-exclude markdown="1">



URI: [ctx:last_commit](https://example.org/it-intelligence/last_commit)
<!-- no inheritance hierarchy -->





## Applicable Classes

| Name | Description | Modifies Slot |
| --- | --- | --- |
| [Repository](Repository.md) | A code repository on the git host |  no  |






## Properties

### Type and Range

| Property | Value |
| --- | --- |
| Range | [Datetime](Datetime.md) |
| Domain Of | [Repository](Repository.md) |
### Slot Characteristics

| Property | Value |
| --- | --- |
| Owner | [Repository](Repository.md) |












## Identifier and Mapping Information





### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:last_commit |
| native | ctx:last_commit |




## LinkML Source

<details>
```yaml
name: last_commit
from_schema: https://example.org/it-intelligence/context-model
rank: 1000
owner: Repository
domain_of:
- Repository
range: datetime

```
</details></div>