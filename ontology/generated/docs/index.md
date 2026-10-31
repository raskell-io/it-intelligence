# The Context Model

Entities concluded from what the sources of an IT landscape say, and the relationships between them. Every relationship is stored as a row with a basis, a confidence and a source.

URI: https://example.org/it-intelligence/context-model

Name: context_model



## Classes

| Class | Description |
| --- | --- |
| [Entity](Entity.md) | Something the Context Model concluded exists, from one or more observations |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Application](Application.md) | A thing the business recognises as one system, however many parts it has |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Capability](Capability.md) | A business capability from the enterprise architecture capability map |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Decision](Decision.md) | An architecture decision record, with the entities it constrains |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Endpoint](Endpoint.md) | A reachable host and path prefix |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[MarketUnit](MarketUnit.md) | An organisational unit of the group, usually a country or a region |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Namespace](Namespace.md) | A namespace on a container cluster |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Repository](Repository.md) | A code repository on the git host |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Team](Team.md) | An accountable team |
| &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Workload](Workload.md) | A deployed unit on a cluster, such as a deployment, a stateful set or a cron ... |



## Slots

| Slot | Description |
| --- | --- |
| [archived](archived.md) | The git host marks the repository read-only |
| [built_from](built_from.md) | The workload's image is built from this repository |
| [cluster](cluster.md) |  |
| [committers_12m](committers_12m.md) | Distinct committers in the last twelve months |
| [constrains](constrains.md) | The decision applies to this entity |
| [decided_on](decided_on.md) |  |
| [depends_on](depends_on.md) | The application calls, or needs, this other application |
| [deployed_as](deployed_as.md) | The application runs as this workload |
| [description](description.md) |  |
| [ea_id](ea_id.md) | The application id in the EA repository, when it has one |
| [exposed_via](exposed_via.md) | The application is reachable through this endpoint |
| [exposure](exposure.md) |  |
| [id](id.md) | The Context Model id, for example ctx:Application/policy-admin |
| [image](image.md) |  |
| [incidents_90d](incidents_90d.md) | Incidents in the last 90 days, from IT service management |
| [last_commit](last_commit.md) |  |
| [last_seen](last_seen.md) |  |
| [level](level.md) |  |
| [lifecycle](lifecycle.md) |  |
| [name](name.md) |  |
| [not_in_ea](not_in_ea.md) | True when no EA repository record backs the application |
| [operated_by](operated_by.md) | The application is operated by, and serves, this market unit |
| [owned_by](owned_by.md) | The accountable team |
| [part_of](part_of.md) | The team belongs to this market unit |
| [realised_by](realised_by.md) | The application is implemented in this repository |
| [refines](refines.md) | The capability is a part of this broader capability |
| [replaced_by](replaced_by.md) | The application is, or will be, replaced by this successor |
| [replicas](replicas.md) |  |
| [routes_to](routes_to.md) | The endpoint sends traffic to this workload |
| [runs_in](runs_in.md) | The workload runs in this namespace |
| [standing](standing.md) |  |
| [status](status.md) |  |
| [supersedes](supersedes.md) | The decision replaces this earlier decision |
| [supports](supports.md) | The application supports this business capability |
| [truck_factor](truck_factor.md) | People who would have to leave before the repository stalls |
| [url](url.md) |  |
| [waf](waf.md) |  |
| [workload_type](workload_type.md) |  |


## Enumerations

| Enumeration | Description |
| --- | --- |
| [Basis](Basis.md) | How a relationship was established |
| [DecisionStatus](DecisionStatus.md) |  |
| [Exposure](Exposure.md) |  |
| [Lifecycle](Lifecycle.md) |  |
| [Standing](Standing.md) | Who is authoritative for a fact |
| [WafMode](WafMode.md) |  |


## Types

| Type | Description |
| --- | --- |
| [Boolean](Boolean.md) | A binary (true or false) value |
| [Curie](Curie.md) | a compact URI |
| [Date](Date.md) | a date (year, month and day) in an idealized calendar |
| [DateOrDatetime](DateOrDatetime.md) | Either a date or a datetime |
| [Datetime](Datetime.md) | The combination of a date and time |
| [Decimal](Decimal.md) | A real number with arbitrary precision that conforms to the xsd:decimal speci... |
| [Double](Double.md) | A real number that conforms to the xsd:double specification |
| [Float](Float.md) | A real number that conforms to the xsd:float specification |
| [Integer](Integer.md) | An integer |
| [Jsonpath](Jsonpath.md) | A string encoding a JSON Path |
| [Jsonpointer](Jsonpointer.md) | A string encoding a JSON Pointer |
| [Ncname](Ncname.md) | Prefix part of CURIE |
| [Nodeidentifier](Nodeidentifier.md) | A URI, CURIE or BNODE that represents a node in a model |
| [Objectidentifier](Objectidentifier.md) | A URI or CURIE that represents an object in the model |
| [Sparqlpath](Sparqlpath.md) | A string encoding a SPARQL Property Path |
| [String](String.md) | A character string |
| [Time](Time.md) | A time object represents a (local) time of day, independent of any particular... |
| [Uri](Uri.md) | a complete URI |
| [Uriorcurie](Uriorcurie.md) | a URI or a CURIE |


## Subsets

| Subset | Description |
| --- | --- |
