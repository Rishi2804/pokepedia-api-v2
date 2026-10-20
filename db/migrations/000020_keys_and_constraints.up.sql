-- Primary keys and unique names for the tables that never had them.
--
-- The item tables were built with inline primary keys, a UNIQUE slug and
-- NOT NULL where the data allows; the older tables relied on convention.
-- This brings them up to the same standard. Every constraint below was
-- checked against the live data first: no duplicate keys, no duplicate names
-- and no NULLs in the columns made NOT NULL.
--
-- abilitydetails: `hidden` is part of the key. 15 Pokemon list the same
-- ability in a normal slot and again as their hidden ability (Great Tusk and
-- Protosynthesis, Palafin and Zero to Hero, ...), so (pokemon_id, ability_id)
-- alone is not unique.
--
-- movedetails: the five-column key is what 000009 and 000012 already guarded
-- with WHERE NOT EXISTS, so those inserts become ON CONFLICT in the scraper.
-- Adding the key makes `version` NOT NULL; it never was NULL.
--
-- species.name was nullable but never NULL.
--
-- The plain btree name indexes are dropped: each UNIQUE constraint builds an
-- equivalent one. The trigram indexes (000005) stay, and so does nothing else
-- that is not duplicated: item_showdown_id's own index duplicates its UNIQUE.

BEGIN;

ALTER TABLE public.abilitydetails
    ADD CONSTRAINT abilitydetails_pkey PRIMARY KEY (pokemon_id, ability_id, hidden);

ALTER TABLE public.movedetails
    ADD CONSTRAINT movedetails_pkey PRIMARY KEY (pokemon_id, move_id, method, level_learned, version);

ALTER TABLE public.pastmovevalues
    ADD CONSTRAINT pastmovevalues_pkey PRIMARY KEY (id, version_groups);

ALTER TABLE public.pasttypes
    ADD CONSTRAINT pasttypes_pkey PRIMARY KEY (pokemon_id, gen);

ALTER TABLE public.dexnumber
    ADD CONSTRAINT dexnumber_pkey PRIMARY KEY (name, species_id, default_variate);

ALTER TABLE public.species ALTER COLUMN name SET NOT NULL;

DROP INDEX IF EXISTS public.idx_pokemon_name;
DROP INDEX IF EXISTS public.idx_move_name;
DROP INDEX IF EXISTS public.idx_ability_name;
DROP INDEX IF EXISTS public.idx_species_name;
DROP INDEX IF EXISTS public.idx_item_name;
DROP INDEX IF EXISTS public.idx_item_showdown_id;

ALTER TABLE public.pokemon ADD CONSTRAINT pokemon_name_key UNIQUE (name);
ALTER TABLE public.move    ADD CONSTRAINT move_name_key    UNIQUE (name);
ALTER TABLE public.ability ADD CONSTRAINT ability_name_key UNIQUE (name);
ALTER TABLE public.species ADD CONSTRAINT species_name_key UNIQUE (name);
ALTER TABLE public.item    ADD CONSTRAINT item_name_key    UNIQUE (name);

COMMIT;
