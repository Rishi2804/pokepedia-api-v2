CREATE TYPE public.dex AS ENUM (
    'national',
    'kanto',
    'original-johto',
    'hoenn',
    'original-sinnoh',
    'extended-sinnoh',
    'updated-johto',
    'original-unova',
    'updated-unova',
    'kalos-central',
    'kalos-coastal',
    'kalos-mountain',
    'updated-hoenn',
    'original-alola',
    'original-melemele',
    'original-akala',
    'original-ulaula',
    'original-poni',
    'updated-alola',
    'updated-melemele',
    'updated-akala',
    'updated-ulaula',
    'updated-poni',
    'letsgo-kanto',
    'galar',
    'isle-of-armor',
    'crown-tundra',
    'hisui',
    'paldea',
    'kitakami',
    'blueberry'
);
CREATE DOMAIN public.generation AS integer
	CONSTRAINT generation_check CHECK (((VALUE >= 1) AND (VALUE <= 12)));
CREATE TYPE public."group" AS ENUM (
    'red-blue',
    'yellow',
    'gold-silver',
    'crystal',
    'ruby-sapphire',
    'emerald',
    'firered-leafgreen',
    'diamond-pearl',
    'platinum',
    'heartgold-soulsilver',
    'black-white',
    'black-2-white-2',
    'x-y',
    'omega-ruby-alpha-sapphire',
    'sun-moon',
    'ultra-sun-ultra-moon',
    'lets-go-pikachu-lets-go-eevee',
    'sword-shield',
    'brilliant-diamond-and-shining-pearl',
    'legends-arceus',
    'scarlet-violet'
);
CREATE TYPE public.learnmethod AS ENUM (
    'level-up',
    'egg',
    'tutor',
    'machine',
    'light-ball-egg',
    'form-change',
    'zygarde-cube'
);
CREATE TYPE public.mclass AS ENUM (
    'physical',
    'special',
    'status'
);
CREATE TYPE public.ptype AS ENUM (
    'fire',
    'water',
    'grass',
    'normal',
    'fighting',
    'ghost',
    'electric',
    'ground',
    'rock',
    'steel',
    'psychic',
    'dark',
    'bug',
    'fairy',
    'flying',
    'poison',
    'ice',
    'dragon',
    'stellar'
);
CREATE DOMAIN public.stat AS integer NOT NULL
	CONSTRAINT stat_check CHECK (((VALUE >= 1) AND (VALUE <= 255)));
CREATE TABLE public.evolution (
    chain_id integer NOT NULL,
    id integer NOT NULL,
    from_pokemon integer NOT NULL,
    from_display text NOT NULL,
    to_pokemon integer NOT NULL,
    to_display text NOT NULL,
    details text[] NOT NULL,
    region text,
    alt_form integer NOT NULL,
    next_evo integer[] NOT NULL,
    prev_evo integer[] NOT NULL
);
CREATE FUNCTION public.get_evolution_chain_by_id(p_id integer) RETURNS SETOF public.evolution
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY
    WITH chain_id_from_evolution AS (
        SELECT e.chain_id
        FROM evolution e
        WHERE e.from_pokemon = p_id OR e.to_pokemon = p_id
        LIMIT 1
    ),
    chain_id_from_evolutionpeek AS (
        SELECT ep.chain_id
        FROM evolutionpeek ep
        WHERE ep.pokemon_id = p_id
        LIMIT 1
    )
    
    SELECT e.*
    FROM evolution e
    WHERE e.chain_id IN (
        SELECT chain_id FROM chain_id_from_evolution
        UNION
        SELECT chain_id FROM chain_id_from_evolutionpeek
    );
END;
$$;
ALTER FUNCTION public.get_evolution_chain_by_id(p_id integer) OWNER TO postgres;
CREATE FUNCTION public.get_evolution_chain_by_name(pokemon_name text) RETURNS SETOF public.evolution
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY
    WITH pokemon_id AS (
        SELECT p.id
        FROM pokemon p
        WHERE p.name = pokemon_name
    ),
    chain_id_from_evolution AS (
        SELECT e.chain_id
        FROM evolution e
        WHERE e.from_pokemon IN (SELECT id FROM pokemon_id)
           OR e.to_pokemon IN (SELECT id FROM pokemon_id)
        LIMIT 1
    ),
    chain_id_from_evolutionpeek AS (
        SELECT ep.chain_id
        FROM evolutionpeek ep
        WHERE ep.pokemon_id IN (SELECT id FROM pokemon_id)
        LIMIT 1
    )
    
    SELECT e.*
    FROM evolution e
    WHERE e.chain_id IN (
        SELECT chain_id FROM chain_id_from_evolution
        UNION
        SELECT chain_id FROM chain_id_from_evolutionpeek
    );
END;
$$;
ALTER FUNCTION public.get_evolution_chain_by_name(pokemon_name text) OWNER TO postgres;
CREATE FUNCTION public.get_pokemon_moves(p_pokemon_id integer) RETURNS TABLE(move_id integer, name text, type public.ptype, level_learned integer, method public.learnmethod, version public."group", class public.mclass, power integer, accuracy integer, pp integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
  RETURN QUERY
  SELECT 
    d.move_id,
    m.name,
    m.type,
	d.level_learned,
    d.method,
    d.version,
    m.class,
    CASE 
      WHEN pmd.id IS NOT NULL AND pmd.version_groups @> ARRAY[d.version] THEN pmd.power
      ELSE m.power 
    END AS power,
    CASE 
      WHEN pmd.id IS NOT NULL AND pmd.version_groups @> ARRAY[d.version] THEN pmd.accuracy
      ELSE m.accuracy 
    END AS accuracy,
    CASE 
      WHEN pmd.id IS NOT NULL AND pmd.version_groups @> ARRAY[d.version] THEN pmd.pp
      ELSE m.pp 
    END AS pp
  FROM movedetails d
  JOIN move m ON d.move_id = m.id
  LEFT JOIN pastmovevalues pmd ON d.move_id = pmd.id AND pmd.version_groups @> ARRAY[d.version]
  WHERE d.pokemon_id = p_pokemon_id;
END; $$;
ALTER FUNCTION public.get_pokemon_moves(p_pokemon_id integer) OWNER TO postgres;
CREATE TABLE public.ability (
    id integer NOT NULL,
    name text NOT NULL,
    gen public.generation NOT NULL,
    descriptions jsonb[] NOT NULL,
    effect text
);
CREATE TABLE public.abilitydetails (
    pokemon_id integer NOT NULL,
    ability_id integer NOT NULL,
    hidden boolean NOT NULL,
    gen public.generation
);
CREATE TABLE public.dexentries (
    pokemon_id integer NOT NULL,
    game text NOT NULL,
    entry text NOT NULL
);
CREATE TABLE public.dexnumber (
    species_id integer NOT NULL,
    name public.dex NOT NULL,
    num integer NOT NULL,
    default_variate integer NOT NULL,
    alt_variates integer[] DEFAULT '{}'::integer[] NOT NULL
);
CREATE TABLE public.evolutionpeek (
    chain_id integer NOT NULL,
    pokemon_id integer NOT NULL
);
CREATE TABLE public.move (
    id integer NOT NULL,
    name text NOT NULL,
    gen public.generation NOT NULL,
    type public.ptype NOT NULL,
    class public.mclass NOT NULL,
    power integer,
    accuracy integer,
    pp integer,
    descriptions jsonb[] NOT NULL,
    effect text
);
CREATE TABLE public.movedetails (
    pokemon_id integer NOT NULL,
    move_id integer NOT NULL,
    method public.learnmethod NOT NULL,
    level_learned integer NOT NULL,
    version public."group"
);
CREATE TABLE public.pastmovevalues (
    id integer NOT NULL,
    power integer,
    accuracy integer,
    pp integer,
    version_groups public."group"[] NOT NULL
);
CREATE TABLE public.pasttypes (
    pokemon_id integer NOT NULL,
    type1 public.ptype NOT NULL,
    type2 public.ptype,
    gen public.generation NOT NULL
);
CREATE TABLE public.pokemon (
    id integer NOT NULL,
    name text NOT NULL,
    gen public.generation NOT NULL,
    type1 public.ptype NOT NULL,
    type2 public.ptype,
    weight numeric(10,2) NOT NULL,
    height numeric(5,1) NOT NULL,
    gender_rate integer NOT NULL,
    hp public.stat NOT NULL,
    atk public.stat NOT NULL,
    def public.stat NOT NULL,
    spatk public.stat NOT NULL,
    spdef public.stat NOT NULL,
    speed public.stat NOT NULL,
    bst integer NOT NULL,
    dex_entries jsonb[] NOT NULL,
    species_id integer NOT NULL,
    forms text[],
    CONSTRAINT check_bst CHECK ((bst = ((((((hp)::integer + (atk)::integer) + (def)::integer) + (spatk)::integer) + (spdef)::integer) + (speed)::integer))),
    CONSTRAINT pokemon_gender_rate_check CHECK (((gender_rate >= '-1'::integer) AND (gender_rate <= 8)))
);
CREATE TABLE public.species (
    id integer NOT NULL,
    name text
);

ALTER TABLE ONLY public.abilitydetails
    ADD CONSTRAINT abilitydetails_ability_id_fkey FOREIGN KEY (ability_id) REFERENCES public.ability(id);
ALTER TABLE ONLY public.abilitydetails
    ADD CONSTRAINT abilitydetails_pokemon_id_fkey FOREIGN KEY (pokemon_id) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.dexentries
    ADD CONSTRAINT dexentries_pokemon_id_fkey FOREIGN KEY (pokemon_id) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.dexnumber
    ADD CONSTRAINT dexnumbers_default_variate_fkey FOREIGN KEY (default_variate) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.dexnumber
    ADD CONSTRAINT dexnumbers_species_id_fkey FOREIGN KEY (species_id) REFERENCES public.species(id);
ALTER TABLE ONLY public.evolution
    ADD CONSTRAINT evolution_from_pokemon_fkey FOREIGN KEY (from_pokemon) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.evolution
    ADD CONSTRAINT evolution_to_pokemon_fkey FOREIGN KEY (to_pokemon) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.evolutionpeek
    ADD CONSTRAINT evolutionpeek_pokemon_id_fkey FOREIGN KEY (pokemon_id) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.movedetails
    ADD CONSTRAINT movedetails_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.move(id);
ALTER TABLE ONLY public.movedetails
    ADD CONSTRAINT movedetails_pokemon_id_fkey FOREIGN KEY (pokemon_id) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.pastmovevalues
    ADD CONSTRAINT pastmovevalues_id_fkey FOREIGN KEY (id) REFERENCES public.move(id);
ALTER TABLE ONLY public.pasttypes
    ADD CONSTRAINT pasttypes_pokemon_id_fkey FOREIGN KEY (pokemon_id) REFERENCES public.pokemon(id);
ALTER TABLE ONLY public.pokemon
    ADD CONSTRAINT pokemon_species_id_fkey FOREIGN KEY (species_id) REFERENCES public.species(id);