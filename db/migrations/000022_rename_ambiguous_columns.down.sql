-- Reverse of 000022.

BEGIN;

ALTER INDEX public.idx_dexnumber_dex RENAME TO idx_dexnumber_name;
ALTER TABLE public.dexnumber RENAME COLUMN dex TO name;

ALTER TABLE public.evolution RENAME COLUMN to_slug TO to_display;
ALTER TABLE public.evolution RENAME COLUMN from_slug TO from_display;

ALTER INDEX public.idx_pastmovevalues_move_id RENAME TO idx_pastmovevalues_id;
ALTER TABLE public.pastmovevalues RENAME COLUMN move_id TO id;

COMMIT;
