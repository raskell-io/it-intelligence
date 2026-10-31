---
search:
  boost: 10.0
---

# Class: Endpoint 


_A reachable host and path prefix._



<div data-search-exclude markdown="1">



URI: [ctx:Endpoint](https://example.org/it-intelligence/Endpoint)





```mermaid
 classDiagram
    class Endpoint
    click Endpoint href "../Endpoint/"
      Entity <|-- Endpoint
        click Entity href "../Entity/"
      
      Endpoint : exposure
        
          
    
        
        
        Endpoint --> "0..1" Exposure : exposure
        click Exposure href "../Exposure/"
    

        
      Endpoint : id
        
      Endpoint : last_seen
        
      Endpoint : name
        
      Endpoint : owned_by
        
          
    
        
        
        Endpoint --> "0..1" Team : owned_by
        click Team href "../Team/"
    

        
      Endpoint : routes_to
        
          
    
        
        
        Endpoint --> "*" Workload : routes_to
        click Workload href "../Workload/"
    

        
      Endpoint : standing
        
          
    
        
        
        Endpoint --> "0..1" Standing : standing
        click Standing href "../Standing/"
    

        
      Endpoint : waf
        
          
    
        
        
        Endpoint --> "0..1" WafMode : waf
        click WafMode href "../WafMode/"
    

        
      
```





## Inheritance
* [Entity](Entity.md)
    * **Endpoint**


## Slots

| Name | Cardinality and Range | Description | Inheritance |
| ---  | --- | --- | --- |
| [routes_to](routes_to.md) | * <br/> [Workload](Workload.md) | The endpoint sends traffic to this workload | direct |
| [owned_by](owned_by.md) | 0..1 <br/> [Team](Team.md) | The accountable team | direct |
| [exposure](exposure.md) | 0..1 <br/> [Exposure](Exposure.md) |  | direct |
| [waf](waf.md) | 0..1 <br/> [WafMode](WafMode.md) |  | direct |
| [id](id.md) | 1 <br/> [String](String.md) | The Context Model id, for example ctx:Application/policy-admin | [Entity](Entity.md) |
| [name](name.md) | 1 <br/> [String](String.md) |  | [Entity](Entity.md) |
| [standing](standing.md) | 0..1 <br/> [Standing](Standing.md) |  | [Entity](Entity.md) |
| [last_seen](last_seen.md) | 0..1 <br/> [Datetime](Datetime.md) |  | [Entity](Entity.md) |





## Usages

| used by | used in | type | used |
| ---  | --- | --- | --- |
| [Application](Application.md) | [exposed_via](exposed_via.md) | range | [Endpoint](Endpoint.md) |
| [Endpoint](Endpoint.md) | [routes_to](routes_to.md) | domain | [Endpoint](Endpoint.md) |












## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_id_pattern | {host}{path} |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:Endpoint |
| native | ctx:Endpoint |






## LinkML Source

<!-- TODO: investigate https://stackoverflow.com/questions/37606292/how-to-create-tabbed-code-blocks-in-mkdocs-or-sphinx -->

### Direct

<details>
```yaml
name: Endpoint
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{host}{path}'
description: A reachable host and path prefix.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
slots:
- routes_to
- owned_by
attributes:
  exposure:
    name: exposure
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    domain_of:
    - Endpoint
    range: Exposure
  waf:
    name: waf
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    domain_of:
    - Endpoint
    range: WafMode

```
</details>

### Induced

<details>
```yaml
name: Endpoint
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{host}{path}'
description: A reachable host and path prefix.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
attributes:
  exposure:
    name: exposure
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    owner: Endpoint
    domain_of:
    - Endpoint
    range: Exposure
  waf:
    name: waf
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    owner: Endpoint
    domain_of:
    - Endpoint
    range: WafMode
  routes_to:
    name: routes_to
    annotations:
      ctx_domain:
        tag: ctx_domain
        value: Endpoint
      ctx_range:
        tag: ctx_range
        value: Workload
      ctx_cardinality:
        tag: ctx_cardinality
        value: many
    description: The endpoint sends traffic to this workload.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    domain: Endpoint
    owner: Endpoint
    domain_of:
    - Endpoint
    range: Workload
    multivalued: true
  owned_by:
    name: owned_by
    annotations:
      ctx_domain:
        tag: ctx_domain
        value: Application Repository Workload Namespace Endpoint
      ctx_range:
        tag: ctx_range
        value: Team
      ctx_cardinality:
        tag: ctx_cardinality
        value: one
    description: The accountable team.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Endpoint
    domain_of:
    - Application
    - Repository
    - Workload
    - Namespace
    - Endpoint
    range: Team
  id:
    name: id
    description: The Context Model id, for example ctx:Application/policy-admin.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    identifier: true
    owner: Endpoint
    domain_of:
    - Entity
    range: string
    required: true
  name:
    name: name
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Endpoint
    domain_of:
    - Entity
    range: string
    required: true
  standing:
    name: standing
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Endpoint
    domain_of:
    - Entity
    range: Standing
  last_seen:
    name: last_seen
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Endpoint
    domain_of:
    - Entity
    range: datetime

```
</details></div>