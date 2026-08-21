-- Type evolution.region as an enum and give evolutionpeek the primary key it
-- never had (it already has a FK on pokemon_id). chain_id/next_evo/prev_evo
-- are intentionally left untouched — evolution's row type is unchanged, so
-- nothing that reads get_evolution_chain_by_id()/_by_name() needs updating.

BEGIN;

-- region held a closed 4-value set as plain text (488 of 526 rows NULL),
-- the same untyped-slug problem 000003 fixed for dexentries.game.
CREATE TYPE public.region AS ENUM ('alola', 'galar', 'hisui', 'paldea');

ALTER TABLE public.evolution
    ALTER COLUMN region TYPE public.region USING region::public.region;

-- evolutionpeek already has a FK on pokemon_id (000001_baseline.up.sql:305)
-- but no primary key. It has one duplicate row (toxtricity-gmax, chain_id
-- 446) that a PK would reject. It's invisible today only because
-- get_evolution_chain_by_id() reads this table with LIMIT 1 — exactly the
-- kind of bug a PK exists to prevent. Keep the lowest ctid, drop the rest.
DELETE FROM public.evolutionpeek ep
WHERE ep.ctid <> (
    SELECT min(dup.ctid) FROM public.evolutionpeek dup
    WHERE dup.chain_id = ep.chain_id AND dup.pokemon_id = ep.pokemon_id
);

ALTER TABLE public.evolutionpeek
    ADD CONSTRAINT evolutionpeek_pkey PRIMARY KEY (chain_id, pokemon_id);

-- No FK on chain_id: there is no chain table to reference, and
-- evolution.chain_id is a grouping key, not unique, so it can't be a target.

-- The PK's (chain_id, pokemon_id) index supersedes this one (same leading
-- column). idx_evolutionpeek_pokemon_id stays — it backs the
-- `WHERE ep.pokemon_id = p_id` lookup in get_evolution_chain_by_id(), which
-- the PK's column order doesn't serve.
DROP INDEX IF EXISTS public.idx_evolutionpeek_chain_id;

COMMIT;
