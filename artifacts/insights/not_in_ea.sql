-- name: not_in_ea
-- severity: warning
-- description: Applications running on the clusters that the EA repository does not know.
select array[a.id] as entities,
       a.name || ' runs, but the EA repository has no record of it' as message
from entity a
where a.kind = 'Application' and coalesce((a.attributes->>'not_in_ea')::boolean, false)
order by a.id;
