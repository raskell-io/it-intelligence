-- name: namespace_without_owner
-- severity: warning
-- description: Namespaces nobody owns, by label or by what runs in them.
select array[n.id] as entities,
       n.name || ' has no owner label and nothing in it tells who owns it' as message
from entity n
where n.kind = 'Namespace'
  and not exists (select 1 from edge e where e.from_id = n.id and e.kind = 'owned_by' and e.status = 'active')
order by n.id;
