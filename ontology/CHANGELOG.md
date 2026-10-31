# Context Model changelog

Every change to an entity kind or relation kind, with the question it lets someone answer.

## 0.1.0 (2026-10-31)

Nine entity kinds: Application, Repository, Workload, Namespace, Team, Capability, Endpoint, Decision,
MarketUnit. Fifteen relation kinds: realised_by, deployed_as, runs_in, built_from, owned_by, part_of,
operated_by, supports, refines, exposed_via, routes_to, depends_on, constrains, supersedes, replaced_by.

The questions they answer: who owns this; what implements and runs an application; what breaks if it
changes; which applications support a capability in each market unit; what is exposed to the internet and
how; which architecture decisions constrain a system; what replaces what.

Deliberately not kinds yet: business service (resolves to Application), incident (an attribute rollup),
WAF policy (an Endpoint attribute), cluster (a Namespace attribute), data object.
