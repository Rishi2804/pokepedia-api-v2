-- Reverse of 000006 -- deliberately a no-op.
--
-- Postgres has no ALTER TYPE ... DROP VALUE. Removing these four labels would
-- mean recreating public.game, public."group" and public.dex from scratch and
-- re-typing every dependent object: dexnumber.name, pokemondescriptions.version,
-- movedescriptions.version, abilitydescriptions.version, movedetails.version,
-- pastmovevalues.version_groups (an enum ARRAY), and get_pokemon_moves(), whose
-- RETURNS TABLE signature names two of them and which would have to be dropped
-- and recreated. That is a large, high-blast-radius rewrite to remove labels
-- that are inert once 000007's down has deleted every row referencing them.
--
-- Same reasoning as 000005 declining to DROP EXTENSION pg_trgm. 000006's up is
-- written with IF NOT EXISTS so a subsequent up replays cleanly.

BEGIN;
COMMIT;
