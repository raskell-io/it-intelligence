---
search:
  boost: 10.0
---

# Class: Workload 


_A deployed unit on a cluster, such as a deployment, a stateful set or a cron job._



<div data-search-exclude markdown="1">



URI: [ctx:Workload](https://example.org/it-intelligence/Workload)





```mermaid
 classDiagram
    class Workload
    click Workload href "../Workload/"
      Entity <|-- Workload
        click Entity href "../Entity/"
      
      Workload : built_from
        
          
    
        
        
        Workload --> "0..1" Repository : built_from
        click Repository href "../Repository/"
    

        
      Workload : id
        
      Workload : image
        
      Workload : last_seen
        
      Workload : name
        
      Workload : owned_by
        
          
    
        
        
        Workload --> "0..1" Team : owned_by
        click Team href "../Team/"
    

        
      Workload : replicas
        
      Workload : runs_in
        
          
    
        
        
        Workload --> "0..1" Namespace : runs_in
        click Namespace href "../Namespace/"
    

        
      Workload : standing
        
          
    
        
        
        Workload --> "0..1" Standing : standing
        click Standing href "../Standing/"
    

        
      Workload : workload_type
        
      
```





## Inheritance
* [Entity](Entity.md)
    * **Workload**


## Slots

| Name | Cardinality and Range | Description | Inheritance |
| ---  | --- | --- | --- |
| [runs_in](runs_in.md) | 0..1 <br/> [Namespace](Namespace.md) | The workload runs in this namespace | direct |
| [built_from](built_from.md) | 0..1 <br/> [Repository](Repository.md) | The workload's image is built from this repository | direct |
| [owned_by](owned_by.md) | 0..1 <br/> [Team](Team.md) | The accountable team | direct |
| [workload_type](workload_type.md) | 0..1 <br/> [String](String.md) |  | direct |
| [replicas](replicas.md) | 0..1 <br/> [Integer](Integer.md) |  | direct |
| [image](image.md) | 0..1 <br/> [String](String.md) |  | direct |
| [id](id.md) | 1 <br/> [String](String.md) | The Context Model id, for example ctx:Application/policy-admin | [Entity](Entity.md) |
| [name](name.md) | 1 <br/> [String](String.md) |  | [Entity](Entity.md) |
| [standing](standing.md) | 0..1 <br/> [Standing](Standing.md) |  | [Entity](Entity.md) |
| [last_seen](last_seen.md) | 0..1 <br/> [Datetime](Datetime.md) |  | [Entity](Entity.md) |





## Usages

| used by | used in | type | used |
| ---  | --- | --- | --- |
| [Application](Application.md) | [deployed_as](deployed_as.md) | range | [Workload](Workload.md) |
| [Workload](Workload.md) | [runs_in](runs_in.md) | domain | [Workload](Workload.md) |
| [Workload](Workload.md) | [built_from](built_from.md) | domain | [Workload](Workload.md) |
| [Endpoint](Endpoint.md) | [routes_to](routes_to.md) | range | [Workload](Workload.md) |












## Identifier and Mapping Information



### Annotations

| property | value |
| --- | --- |
| ctx_id_pattern | {namespace}/{name} |




### Schema Source


* from schema: https://example.org/it-intelligence/context-model




## Mappings

| Mapping Type | Mapped Value |
| ---  | ---  |
| self | ctx:Workload |
| native | ctx:Workload |






## LinkML Source

<!-- TODO: investigate https://stackoverflow.com/questions/37606292/how-to-create-tabbed-code-blocks-in-mkdocs-or-sphinx -->

### Direct

<details>
```yaml
name: Workload
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{namespace}/{name}'
description: A deployed unit on a cluster, such as a deployment, a stateful set or
  a cron job.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
slots:
- runs_in
- built_from
- owned_by
attributes:
  workload_type:
    name: workload_type
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    domain_of:
    - Workload
  replicas:
    name: replicas
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    domain_of:
    - Workload
    range: integer
  image:
    name: image
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    domain_of:
    - Workload

```
</details>

### Induced

<details>
```yaml
name: Workload
annotations:
  ctx_id_pattern:
    tag: ctx_id_pattern
    value: '{namespace}/{name}'
description: A deployed unit on a cluster, such as a deployment, a stateful set or
  a cron job.
from_schema: https://example.org/it-intelligence/context-model
is_a: Entity
attributes:
  workload_type:
    name: workload_type
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    owner: Workload
    domain_of:
    - Workload
    range: string
  replicas:
    name: replicas
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    owner: Workload
    domain_of:
    - Workload
    range: integer
  image:
    name: image
    from_schema: https://example.org/it-intelligence/entities
    rank: 1000
    owner: Workload
    domain_of:
    - Workload
    range: string
  runs_in:
    name: runs_in
    annotations:
      ctx_domain:
        tag: ctx_domain
        value: Workload
      ctx_range:
        tag: ctx_range
        value: Namespace
      ctx_cardinality:
        tag: ctx_cardinality
        value: one
    description: The workload runs in this namespace.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    domain: Workload
    owner: Workload
    domain_of:
    - Workload
    range: Namespace
  built_from:
    name: built_from
    annotations:
      ctx_domain:
        tag: ctx_domain
        value: Workload
      ctx_range:
        tag: ctx_range
        value: Repository
      ctx_cardinality:
        tag: ctx_cardinality
        value: one
    description: The workload's image is built from this repository.
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    domain: Workload
    owner: Workload
    domain_of:
    - Workload
    range: Repository
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
    owner: Workload
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
    owner: Workload
    domain_of:
    - Entity
    range: string
    required: true
  name:
    name: name
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Workload
    domain_of:
    - Entity
    range: string
    required: true
  standing:
    name: standing
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Workload
    domain_of:
    - Entity
    range: Standing
  last_seen:
    name: last_seen
    from_schema: https://example.org/it-intelligence/context-model
    rank: 1000
    owner: Workload
    domain_of:
    - Entity
    range: datetime

```
</details></div>