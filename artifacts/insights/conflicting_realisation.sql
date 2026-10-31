-- name: conflicting_realisation
-- severity: warning
-- description: Repositories that more than one application claims.
select array[r.id] as entities,
       r.name || ' is claimed by ' || string_agg(distinct a.name, ' and ' order by a.name) as message
from entity r
join edge e on e.to_id = r.id and e.kind = 'realised_by' and e.status = 'active'
join entity a on a.id = e.from_id
where r.kind = 'Repository'
group by r.id, r.name
having count(distinct a.id) > 1
order by r.id;
