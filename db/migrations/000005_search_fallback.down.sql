-- Reverse of 000005. pg_trgm is not dropped — other objects may come to
-- depend on it, and DROP EXTENSION would refuse if any do anyway.

BEGIN;

DROP INDEX IF EXISTS public.idx_pokemon_name_trgm;
DROP INDEX IF EXISTS public.idx_move_name_trgm;
DROP INDEX IF EXISTS public.idx_ability_name_trgm;

COMMIT;
