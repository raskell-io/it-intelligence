-- name: conflicting_owner
-- severity: error
-- description: Entities that two sources say different teams own.
select array[x.id] || array_agg(distinct e.to_id order by e.to_id) as entities,
       x.name || ' has ' || count(distinct e.to_id) || ' different owners' as message
from entity x
join edge e on e.from_id = x.id and e.kind = 'owned_by' and e.status = 'active' and e.source <> 'resolver'
group by x.id, x.name
having count(distinct e.to_id) > 1
order by x.id;
