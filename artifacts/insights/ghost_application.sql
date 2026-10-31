-- name: ghost_application
-- severity: warning
-- description: Applications the EA repository lists as in production that no other source has seen: no repository, no workload, no incident.
select array[a.id] as entities,
       a.name || ' is listed as production, but nothing realises or runs it' as message
from entity a
where a.kind = 'Application'
  and a.attributes->>'lifecycle' = 'production'
  and not exists (select 1 from edge e where e.from_id = a.id and e.kind in ('realised_by', 'deployed_as') and e.status = 'active')
  and coalesce((a.attributes->>'incidents_90d')::int, 0) = 0
order by a.id;
