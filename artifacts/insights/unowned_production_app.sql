-- name: unowned_production_app
-- severity: error
-- description: Applications in production with no accountable owner at confidence 0.8 or above.
select array[a.id] as entities,
       a.name || ' is in production and has no owner the model is sure of' as message
from entity a
where a.kind = 'Application'
  and a.attributes->>'lifecycle' = 'production'
  and not exists (
    select 1 from edge e
    where e.from_id = a.id and e.kind = 'owned_by' and e.status = 'active'
      and e.confidence >= 0.8
  )
order by a.id;
