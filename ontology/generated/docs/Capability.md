---
search:
  boost: 10.0
---

# Class: Capability 


_A business capability from the enterprise architecture capability map._



<div data-search-exclude markdown="1">



URI: [ctx:Capability](https://example.org/it-intelligence/Capability)





```mermaid
 classDiagram
    class Capability
    click Capability href "../Capability/"
      Entity <|-- Capability
        click Entity href "../Entity/"
      
      Capability : id
        
      Capability : last_seen
        
      Capability : level
        
      Capability : name
        
      Capability : refines
        
          
    
        
        
        Capability --> "0..1" Capability : refines
        click Capability href "../Capability/"
    

        
      Capability : standing
        
          
    
        
        
        Capability --> "0..1" Standing : standing
        click Standing href "../Standing/"
    

        
      
```





## Inheritance
* [Entity](Entity.md)
    * **Capability**


## Slots

| Name | Cardinality and Range | Description | Inheritance |
| ---  | --- | --- | --- |
| [refines](refines.md) | 0..1 <br/> [Capability](Capability.md) | The capability is a part of this broader capability | direct |
| [level](level.md) | 0..1 <br/> [Integer](Integer.md) |  | direct |
| [id](id.md) | 1 <br/> [String](String.md) | The Context Model id, for example ctx:Application/policy-admin | [Entity](Entity.md) |
| [name](name.md) | 1 <br/> [String](String.md) |  | [Entity](Entity.md) |
| [standing](standing.md) | 0..1 <br/> [Standing](Standing.md) |  | [Entity](Entity.md) |
| [last_seen](last_seen.md) | 0..1 <br/> [Datetime](Datetime.md) |  | [Entity](Entity.md) |





## Usages

| used by | used in | type | used |
| ---  | --- | --- | --- |
| [Application](Application.md) | [supports](supports.md) | range | [Capability](Capability.md) |
| [Capability](Capability.md) | [refines](refines.md) | domain | [Capability](Capability.md) |
| [Capability](Capability.md) | [refines](refines.md) | range | [Capability](Capability.md) |












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
| self | ctx:Capability |
| native | ctx:Capability |






## LinkML Source

<!-- TODO: investigate https://stackoverflow.com/questions/37606292/how-to-create-tabbed-code-blocks-in-mkdocs-or-sphinx -->

### Direct

<details>
```yaml
name: Capability
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{slug}'
description: A business capability from the enterprise architecture capability map.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
slots:
- refines
attributes:
  level:
    name: level
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    domain_of:
    - Capability
    range: integer

```
</details>

### Induced

<details>
```yaml
name: Capability
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{slug}'
description: A business capability from the enterprise architecture capability map.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
attributes:
  level:
    name: level
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    owner: Capability
    domain_of:
    - Capability
    range: integer
  refines:
    name: refines
    annotations:
      ctx_domain:
        tag: ctx_domain
        value: Capability
      ctx_range:
        tag: ctx_range
        value: Capability
      ctx_cardinality:
        tag: ctx_cardinality
        value: one
    description: The capability is a part of this broader capability.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    domain: Capability
    owner: Capability
    domain_of:
    - Capability
    range: Capability
  id:
    name: id
    description: The Context Model id, for example ctx:Application/policy-admin.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    identifier: true
    owner: Capability
    domain_of:
    - Entity
    range: string
    required: true
  name:
    name: name
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Capability
    domain_of:
    - Entity
    range: string
    required: true
  standing:
    name: standing
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Capability
    domain_of:
    - Entity
    range: Standing
  last_seen:
    name: last_seen
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Capability
    domain_of:
    - Entity
    range: datetime

```
</details></div>