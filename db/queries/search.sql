-- name: SearchNamesFallback :many
-- Degraded fallback for when Elasticsearch is down. term is pre-slugified
-- (util.Slugify). Threshold is explicit, not the pg_trgm GUC, since 0.3 is
-- too strict for short slugs. Infix LIKE catches "blast" -> "moonblast".
WITH q AS (
    SELECT sqlc.arg(term)::text AS term
)
SELECT s.type, s.id, s.name, s.gen, s.score
FROM (
    SELECT 'pokemon'::text AS type, p.id, p.name, p.gen::int AS gen,
           GREATEST(similarity(p.name, q.term),
                    CASE WHEN p.name LIKE q.term || '%' THEN 1.0 ELSE 0 END) AS score
    FROM public.pokemon p, q
    WHERE similarity(p.name, q.term) > 0.2
       OR p.name LIKE q.term || '%'
       OR p.name LIKE '%' || q.term || '%'
    UNION ALL
    SELECT 'move', m.id, m.name, m.gen::int,
           GREATEST(similarity(m.name, q.term),
                    CASE WHEN m.name LIKE q.term || '%' THEN 1.0 ELSE 0 END)
    FROM public.move m, q
    WHERE similarity(m.name, q.term) > 0.2
       OR m.name LIKE q.term || '%'
       OR m.name LIKE '%' || q.term || '%'
    UNION ALL
    SELECT 'ability', a.id, a.name, a.gen::int,
           GREATEST(similarity(a.name, q.term),
                    CASE WHEN a.name LIKE q.term || '%' THEN 1.0 ELSE 0 END)
    FROM public.ability a, q
    WHERE similarity(a.name, q.term) > 0.2
       OR a.name LIKE q.term || '%'
       OR a.name LIKE '%' || q.term || '%'
) s
ORDER BY s.score DESC, s.type, s.id
LIMIT sqlc.arg(lim)::int;
