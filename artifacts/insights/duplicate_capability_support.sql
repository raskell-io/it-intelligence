-- name: duplicate_capability_support
-- severity: info
-- description: Capabilities supported by more than one application in the same market unit: consolidation candidates.
select array[c.id, mu.id] || array_agg(distinct a.id order by a.id) as entities,
       c.name || ' in ' || mu.name || ' is supported by ' || count(distinct a.id) || ' applications' as message
from edge s
join entity a on a.id = s.from_id
join entity c on c.id = s.to_id
join edge o on o.from_id = a.id and o.kind = 'operated_by' and o.status = 'active'
join entity mu on mu.id = o.to_id
where s.kind = 'supports' and s.status = 'active'
group by c.id, c.name, mu.id, mu.name
having count(distinct a.id) > 1
order by c.id, mu.id;
