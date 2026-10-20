-- Reverse of the paired .up.sql: drops the constraints it added and lets
-- display_name be NULL again. The values themselves go when 000023's down
-- drops the columns.

BEGIN;

ALTER TABLE public.pokemon DROP CONSTRAINT IF EXISTS pokemon_required_item_id_fkey;
ALTER TABLE public.pokemon DROP CONSTRAINT IF EXISTS pokemon_battle_only_base_id_fkey;
ALTER TABLE public.ability DROP CONSTRAINT IF EXISTS ability_showdown_id_key;
ALTER TABLE public.move DROP CONSTRAINT IF EXISTS move_showdown_id_key;
ALTER TABLE public.pokemon DROP CONSTRAINT IF EXISTS pokemon_showdown_id_key;

ALTER TABLE public.ability ALTER COLUMN display_name DROP NOT NULL;
ALTER TABLE public.move ALTER COLUMN display_name DROP NOT NULL;
ALTER TABLE public.pokemon ALTER COLUMN display_name DROP NOT NULL;
ALTER TABLE public.species ALTER COLUMN display_name DROP NOT NULL;

COMMIT;
