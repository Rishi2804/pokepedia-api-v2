-- Reverse of 000007.
--
-- Ordered by FK dependency: dexnumber, abilitydetails and evolutionpeek all
-- reference pokemon(id); abilitydetails also references ability(id). So the
-- three dependents go first, then pokemon, then ability.
--
-- dexnumber is deleted by dex label rather than by referenced id: alt_variates
-- is a plain integer[] with no foreign key, so there is nothing to cascade from
-- and the label is the only reliable handle on these 364 rows.
--
-- The enum labels added by 000006 survive this -- see 000006_*.down.sql for why
-- that is deliberate and harmless.

BEGIN;

DELETE FROM public.dexnumber      WHERE name IN ('lumiose', 'hyperspace');
DELETE FROM public.evolutionpeek  WHERE pokemon_id BETWEEN 10278 AND 10326;
DELETE FROM public.abilitydetails WHERE pokemon_id BETWEEN 10278 AND 10326;
DELETE FROM public.pokemon        WHERE id         BETWEEN 10278 AND 10326;
DELETE FROM public.ability        WHERE id         BETWEEN 308   AND 313;

COMMIT;
