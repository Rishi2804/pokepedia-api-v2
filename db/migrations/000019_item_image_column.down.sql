-- Reverse of 000019.

BEGIN;

ALTER TABLE public.item RENAME COLUMN image TO sprite;

COMMIT;
