-- Reverse of the paired .up.sql.

BEGIN;

ALTER TABLE public.item DROP COLUMN display_name;

COMMIT;
