-- Trigram support for the degraded (Elasticsearch-down) name search.
--
-- A plain ILIKE cannot serve this path: names are stored as slugs and
-- FormatName (internal/util/name.go) reorders words for display, so what a
-- user types ("Alolan Raichu") shares no prefix, infix, or suffix with the
-- stored slug ("raichu-alola"). Trigram similarity is order-insensitive,
-- which is exactly the property that bridges the reordering.
--
-- pg_trgm ships in postgresql-contrib, already installed in the official
-- postgres:17 image — no Dockerfile change.

BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- pokemon + move + ability is 2,383 rows total, so a sequential scan with
-- similarity() is already sub-millisecond and these indexes are close to
-- decorative today. They're cheap, they make the intent explicit, and they
-- keep this query flat if the corpus grows (items, locations, ...).
CREATE INDEX IF NOT EXISTS idx_pokemon_name_trgm ON public.pokemon USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_move_name_trgm    ON public.move    USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_ability_name_trgm ON public.ability USING gin (name gin_trgm_ops);

COMMIT;
