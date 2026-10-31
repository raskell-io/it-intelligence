-- name: retired_but_running
-- severity: warning
-- description: Applications the EA repository says are retired while workloads still run.
select array[a.id] as entities,
       a.name || ' is retired in the EA repository, but ' || count(*) || ' workload(s) still run' as message
from entity a
join edge e on e.from_id = a.id and e.kind = 'deployed_as' and e.status = 'active'
where a.kind = 'Application' and a.attributes->>'lifecycle' = 'retired'
group by a.id, a.name
order by a.id;
