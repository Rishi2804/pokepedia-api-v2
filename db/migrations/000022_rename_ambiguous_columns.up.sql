-- Rename columns whose names say something other than what they hold.
--
--   pastmovevalues.id        -> move_id   it is a move id, not a row id
--   evolution.from_display   -> from_slug    these hold slugs ("raichu-alola"),
--   evolution.to_display     -> to_slug      not display names; the service
--                                            formats them on the way out
--   dexnumber.name           -> dex       it is a `dex` enum value, not a name
--
-- Primary keys, foreign keys and the evolution row type follow a column
-- rename on their own, so get_evolution_chain_by_id() (SETOF evolution, e.*)
-- needs no change. The two indexes named after the old columns are renamed to
-- match. The API's JSON field names are unchanged.
--
-- Deploy together with the new API build: the old build's queries use the old
-- column names.

BEGIN;

ALTER TABLE public.pastmovevalues RENAME COLUMN id TO move_id;
ALTER INDEX public.idx_pastmovevalues_id RENAME TO idx_pastmovevalues_move_id;

ALTER TABLE public.evolution RENAME COLUMN from_display TO from_slug;
ALTER TABLE public.evolution RENAME COLUMN to_display TO to_slug;

ALTER TABLE public.dexnumber RENAME COLUMN name TO dex;
ALTER INDEX public.idx_dexnumber_name RENAME TO idx_dexnumber_dex;

COMMIT;
