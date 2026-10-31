---
search:
  boost: 10.0
---

# Class: MarketUnit 


_An organisational unit of the group, usually a country or a region._



<div data-search-exclude markdown="1">



URI: [ctx:MarketUnit](https://example.org/it-intelligence/MarketUnit)





```mermaid
 classDiagram
    class MarketUnit
    click MarketUnit href "../MarketUnit/"
      Entity <|-- MarketUnit
        click Entity href "../Entity/"
      
      MarketUnit : id
        
      MarketUnit : last_seen
        
      MarketUnit : name
        
      MarketUnit : standing
        
          
    
        
        
        MarketUnit --> "0..1" Standing : standing
        click Standing href "../Standing/"
    

        
      
```





## Inheritance
* [Entity](Entity.md)
    * **MarketUnit**


## Slots

| Name | Cardinality and Range | Description | Inheritance |
| ---  | --- | --- | --- |
| [id](id.md) | 1 <br/> [String](String.md) | The Context Model id, for example ctx:Application/policy-admin | [Entity](Entity.md) |
| [name](name.md) | 1 <br/> [String](String.md) |  | [Entity](Entity.md) |
| [standing](standing.md) | 0..1 <br/> [Standing](Standing.md) |  | [Entity](Entity.md) |
| [last_seen](last_seen.md) | 0..1 <br/> [Datetime](Datetime.md) |  | [Entity](Entity.md) |





## Usages

| used by | used in | type | used |
| ---  | --- | --- | --- |
| [Application](Application.md) | [operated_by](operated_by.md) | range | [MarketUnit](MarketUnit.md) |
| [Team](Team.md) | [part_of](part_of.md) | range | [MarketUnit](MarketUnit.md) |












## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_id_pattern | {slug} |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:MarketUnit |
| native | ctx:MarketUnit |






## LinkML Source

<!-- TODO: investigate https://stackoverflow.com/questions/37606292/how-to-create-tabbed-code-blocks-in-mkdocs-or-sphinx -->

### Direct

<details>
```yaml
name: MarketUnit
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{slug}'
description: An organisational unit of the group, usually a country or a region.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity

```
</details>

### Induced

<details>
```yaml
name: MarketUnit
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{slug}'
description: An organisational unit of the group, usually a country or a region.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
attributes:
  id:
    name: id
    description: The Context Model id, for example ctx:Application/policy-admin.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    identifier: true
    owner: MarketUnit
    domain_of:
    - Entity
    range: string
    required: true
  name:
    name: name
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: MarketUnit
    domain_of:
    - Entity
    range: string
    required: true
  standing:
    name: standing
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: MarketUnit
    domain_of:
    - Entity
    range: Standing
  last_seen:
    name: last_seen
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: MarketUnit
    domain_of:
    - Entity
    range: datetime

```
</details></div>