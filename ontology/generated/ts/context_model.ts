export type EntityId = string;
export type ApplicationId = string;
export type RepositoryId = string;
export type WorkloadId = string;
export type NamespaceId = string;
export type TeamId = string;
export type CapabilityId = string;
export type EndpointId = string;
export type DecisionId = string;
export type MarketUnitId = string;
/**
* Who is authoritative for a fact. Set per data set; an entity takes the standing of its data sets.
*/
export enum Standing {
    
    /** mirrored from an authoritative system; the source wins */
    secondary = "secondary",
    /** the Context Model is authoritative */
    primary = "primary",
};
/**
* How a relationship was established.
*/
export enum Basis {
    
    /** straight from a source's own key, or confirmed by a person */
    asserted = "asserted",
    /** follows from other relationships by a named rule */
    derived = "derived",
    /** proposed by the resolver with a model's help; carries a confidence and reasons */
    inferred = "inferred",
};

export enum Lifecycle {
    
    planned = "planned",
    production = "production",
    retiring = "retiring",
    retired = "retired",
};

export enum Exposure {
    
    internal = "internal",
    internet = "internet",
};

export enum WafMode {
    
    enforced = "enforced",
    monitor = "monitor",
    none = "none",
};

export enum DecisionStatus {
    
    proposed = "proposed",
    accepted = "accepted",
    superseded = "superseded",
};


/**
 * Something the Context Model concluded exists, from one or more observations.
 */
export interface Entity {
    /** The Context Model id, for example ctx:Application/policy-admin. */
    id: string,
    name: string,
    standing?: string,
    last_seen?: string,
}


/**
 * A thing the business recognises as one system, however many parts it has.
 */
export interface Application extends Entity {
    /** The application is implemented in this repository. */
    realised_by?: RepositoryId[],
    /** The application runs as this workload. */
    deployed_as?: WorkloadId[],
    /** The accountable team. */
    owned_by?: TeamId,
    /** The application is operated by, and serves, this market unit. */
    operated_by?: MarketUnitId[],
    /** The application supports this business capability. */
    supports?: CapabilityId[],
    /** The application is reachable through this endpoint. */
    exposed_via?: EndpointId[],
    /** The application calls, or needs, this other application. */
    depends_on?: ApplicationId[],
    /** The application is, or will be, replaced by this successor. */
    replaced_by?: ApplicationId,
    lifecycle?: string,
    description?: string,
    /** The application id in the EA repository, when it has one. */
    ea_id?: string,
    /** True when no EA repository record backs the application. */
    not_in_ea?: boolean,
    /** Incidents in the last 90 days, from IT service management. */
    incidents_90d?: number,
}


/**
 * A code repository on the git host.
 */
export interface Repository extends Entity {
    /** The accountable team. */
    owned_by?: TeamId,
    description?: string,
    last_commit?: string,
    /** Distinct committers in the last twelve months. */
    committers_12m?: number,
    /** People who would have to leave before the repository stalls. */
    truck_factor?: number,
    /** The git host marks the repository read-only. */
    archived?: boolean,
}


/**
 * A deployed unit on a cluster, such as a deployment, a stateful set or a cron job.
 */
export interface Workload extends Entity {
    /** The workload runs in this namespace. */
    runs_in?: NamespaceId,
    /** The workload's image is built from this repository. */
    built_from?: RepositoryId,
    /** The accountable team. */
    owned_by?: TeamId,
    workload_type?: string,
    replicas?: number,
    image?: string,
}


/**
 * A namespace on a container cluster.
 */
export interface Namespace extends Entity {
    /** The accountable team. */
    owned_by?: TeamId,
    cluster?: string,
}


/**
 * An accountable team.
 */
export interface Team extends Entity {
    /** The team belongs to this market unit. */
    part_of?: MarketUnitId,
}


/**
 * A business capability from the enterprise architecture capability map.
 */
export interface Capability extends Entity {
    /** The capability is a part of this broader capability. */
    refines?: CapabilityId,
    level?: number,
}


/**
 * A reachable host and path prefix.
 */
export interface Endpoint extends Entity {
    /** The endpoint sends traffic to this workload. */
    routes_to?: WorkloadId[],
    /** The accountable team. */
    owned_by?: TeamId,
    exposure?: string,
    waf?: string,
}


/**
 * An architecture decision record, with the entities it constrains.
 */
export interface Decision extends Entity {
    /** The decision applies to this entity. */
    constrains?: EntityId[],
    /** The decision replaces this earlier decision. */
    supersedes?: DecisionId,
    status?: string,
    decided_on?: string,
    url?: string,
}


/**
 * An organisational unit of the group, usually a country or a region.
 */
export interface MarketUnit extends Entity {
}



