-- Reverse of 000023. Run 000024's down first: it drops the constraints that
-- depend on these columns.

BEGIN;

ALTER TABLE public.ability
    DROP COLUMN IF EXISTS showdown_id,
    DROP COLUMN IF EXISTS display_name;

ALTER TABLE public.move
    DROP COLUMN IF EXISTS showdown_id,
    DROP COLUMN IF EXISTS display_name;

ALTER TABLE public.pokemon
    DROP COLUMN IF EXISTS gigantamax,
    DROP COLUMN IF EXISTS required_item_id,
    DROP COLUMN IF EXISTS battle_only_base_id,
    DROP COLUMN IF EXISTS showdown_name,
    DROP COLUMN IF EXISTS showdown_id,
    DROP COLUMN IF EXISTS display_name;

ALTER TABLE public.species DROP COLUMN IF EXISTS display_name;

COMMIT;
