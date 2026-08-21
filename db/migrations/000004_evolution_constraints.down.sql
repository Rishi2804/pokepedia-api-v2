-- Reverse of 000004. The deduped evolutionpeek row (toxtricity-gmax) is not
-- recreated — it was a duplicate with no distinguishing data, so there is
-- nothing to restore it from.

BEGIN;

CREATE INDEX IF NOT EXISTS idx_evolutionpeek_chain_id ON public.evolutionpeek (chain_id);

ALTER TABLE public.evolutionpeek DROP CONSTRAINT evolutionpeek_pkey;
-- evolutionpeek_pokemon_id_fkey predates 000004 (000001_baseline.up.sql) and
-- is left in place.

ALTER TABLE public.evolution
    ALTER COLUMN region TYPE text USING region::text;

DROP TYPE public.region;

COMMIT;
