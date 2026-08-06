-- Secondary indexes. The baseline schema declares foreign keys but Postgres
-- does not auto-index the referencing column, so every WHERE/JOIN predicate
-- below was previously a sequential scan.

CREATE INDEX IF NOT EXISTS idx_movedetails_pokemon_id ON public.movedetails (pokemon_id);
CREATE INDEX IF NOT EXISTS idx_movedetails_move_id ON public.movedetails (move_id);
CREATE INDEX IF NOT EXISTS idx_movedetails_version ON public.movedetails (version);

CREATE INDEX IF NOT EXISTS idx_abilitydetails_pokemon_id ON public.abilitydetails (pokemon_id);
CREATE INDEX IF NOT EXISTS idx_abilitydetails_ability_id ON public.abilitydetails (ability_id);

CREATE INDEX IF NOT EXISTS idx_dexentries_pokemon_id ON public.dexentries (pokemon_id);

CREATE INDEX IF NOT EXISTS idx_pasttypes_pokemon_id ON public.pasttypes (pokemon_id);

CREATE INDEX IF NOT EXISTS idx_pokemon_species_id ON public.pokemon (species_id);
CREATE INDEX IF NOT EXISTS idx_pokemon_name ON public.pokemon (name);

CREATE INDEX IF NOT EXISTS idx_species_name ON public.species (name);
CREATE INDEX IF NOT EXISTS idx_move_name ON public.move (name);
CREATE INDEX IF NOT EXISTS idx_ability_name ON public.ability (name);

CREATE INDEX IF NOT EXISTS idx_dexnumber_name ON public.dexnumber (name);
CREATE INDEX IF NOT EXISTS idx_dexnumber_default_variate ON public.dexnumber (default_variate);
CREATE INDEX IF NOT EXISTS idx_dexnumber_species_id ON public.dexnumber (species_id);
CREATE INDEX IF NOT EXISTS idx_dexnumber_alt_variates ON public.dexnumber USING gin (alt_variates);

CREATE INDEX IF NOT EXISTS idx_pastmovevalues_id ON public.pastmovevalues (id);
CREATE INDEX IF NOT EXISTS idx_pastmovevalues_version_groups ON public.pastmovevalues USING gin (version_groups);

CREATE INDEX IF NOT EXISTS idx_evolution_chain_id ON public.evolution (chain_id);
CREATE INDEX IF NOT EXISTS idx_evolution_from_pokemon ON public.evolution (from_pokemon);
CREATE INDEX IF NOT EXISTS idx_evolution_to_pokemon ON public.evolution (to_pokemon);

CREATE INDEX IF NOT EXISTS idx_evolutionpeek_pokemon_id ON public.evolutionpeek (pokemon_id);
CREATE INDEX IF NOT EXISTS idx_evolutionpeek_chain_id ON public.evolutionpeek (chain_id);
