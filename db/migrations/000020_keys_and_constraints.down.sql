-- Reverse of 000020.

BEGIN;

ALTER TABLE public.item    DROP CONSTRAINT IF EXISTS item_name_key;
ALTER TABLE public.species DROP CONSTRAINT IF EXISTS species_name_key;
ALTER TABLE public.ability DROP CONSTRAINT IF EXISTS ability_name_key;
ALTER TABLE public.move    DROP CONSTRAINT IF EXISTS move_name_key;
ALTER TABLE public.pokemon DROP CONSTRAINT IF EXISTS pokemon_name_key;

CREATE INDEX IF NOT EXISTS idx_pokemon_name ON public.pokemon (name);
CREATE INDEX IF NOT EXISTS idx_move_name ON public.move (name);
CREATE INDEX IF NOT EXISTS idx_ability_name ON public.ability (name);
CREATE INDEX IF NOT EXISTS idx_species_name ON public.species (name);
CREATE INDEX IF NOT EXISTS idx_item_name ON public.item (name);
CREATE INDEX IF NOT EXISTS idx_item_showdown_id ON public.item (showdown_id);

ALTER TABLE public.species ALTER COLUMN name DROP NOT NULL;

ALTER TABLE public.dexnumber DROP CONSTRAINT IF EXISTS dexnumber_pkey;
ALTER TABLE public.pasttypes DROP CONSTRAINT IF EXISTS pasttypes_pkey;
ALTER TABLE public.pastmovevalues DROP CONSTRAINT IF EXISTS pastmovevalues_pkey;

-- Dropping a primary key leaves its columns NOT NULL; version was nullable.
ALTER TABLE public.movedetails DROP CONSTRAINT IF EXISTS movedetails_pkey;
ALTER TABLE public.movedetails ALTER COLUMN version DROP NOT NULL;

ALTER TABLE public.abilitydetails DROP CONSTRAINT IF EXISTS abilitydetails_pkey;

COMMIT;
