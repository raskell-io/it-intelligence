-- name: truck_factor_one
-- severity: warning
-- description: Repositories behind a production application that one person, or nobody, keeps alive.
select array[r.id] as entities,
       r.name || ' realises ' || string_agg(distinct a.name, ', ') || ' and had ' || (r.attributes->>'truck_factor') || ' committer(s) in twelve months' as message
from entity r
join edge e on e.to_id = r.id and e.kind = 'realised_by' and e.status = 'active' and e.confidence >= 0.8
join entity a on a.id = e.from_id and a.attributes->>'lifecycle' = 'production'
where r.kind = 'Repository'
  and coalesce((r.attributes->>'archived')::boolean, false) = false
  and (r.attributes->>'truck_factor')::int <= 1
group by r.id, r.name, r.attributes
order by r.id;
