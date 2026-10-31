-- name: exposed_without_waf
-- severity: error
-- description: Endpoints exposed to the internet without a WAF policy.
select array[ep.id] as entities,
       ep.name || ' is reachable from the internet with no WAF policy' as message
from entity ep
where ep.kind = 'Endpoint'
  and ep.attributes->>'exposure' = 'internet'
  and coalesce(ep.attributes->>'waf', 'none') = 'none'
order by ep.id;
