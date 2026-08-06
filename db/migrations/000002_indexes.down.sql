DROP INDEX IF EXISTS public.idx_movedetails_pokemon_id;
DROP INDEX IF EXISTS public.idx_movedetails_move_id;
DROP INDEX IF EXISTS public.idx_movedetails_version;

DROP INDEX IF EXISTS public.idx_abilitydetails_pokemon_id;
DROP INDEX IF EXISTS public.idx_abilitydetails_ability_id;

DROP INDEX IF EXISTS public.idx_dexentries_pokemon_id;

DROP INDEX IF EXISTS public.idx_pasttypes_pokemon_id;

DROP INDEX IF EXISTS public.idx_pokemon_species_id;
DROP INDEX IF EXISTS public.idx_pokemon_name;

DROP INDEX IF EXISTS public.idx_species_name;
DROP INDEX IF EXISTS public.idx_move_name;
DROP INDEX IF EXISTS public.idx_ability_name;

DROP INDEX IF EXISTS public.idx_dexnumber_name;
DROP INDEX IF EXISTS public.idx_dexnumber_default_variate;
DROP INDEX IF EXISTS public.idx_dexnumber_species_id;
DROP INDEX IF EXISTS public.idx_dexnumber_alt_variates;

DROP INDEX IF EXISTS public.idx_pastmovevalues_id;
DROP INDEX IF EXISTS public.idx_pastmovevalues_version_groups;

DROP INDEX IF EXISTS public.idx_evolution_chain_id;
DROP INDEX IF EXISTS public.idx_evolution_from_pokemon;
DROP INDEX IF EXISTS public.idx_evolution_to_pokemon;

DROP INDEX IF EXISTS public.idx_evolutionpeek_pokemon_id;
DROP INDEX IF EXISTS public.idx_evolutionpeek_chain_id;
