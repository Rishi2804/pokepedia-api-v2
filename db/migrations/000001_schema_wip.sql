CREATE TYPE public.ptype AS ENUM (
    'fire', 'water', 'grass', 'normal', 'fighting', 'ghost', 'electric',
    'ground', 'rock', 'steel', 'psychic', 'dark', 'bug', 'fairy',
    'flying', 'poison', 'ice', 'dragon', 'stellar'
);

CREATE DOMAIN public.stat AS integer NOT NULL
    CONSTRAINT stat_check CHECK (((VALUE >= 1) AND (VALUE <= 255)));

CREATE DOMAIN public.generation AS integer
    CONSTRAINT generation_check CHECK (((VALUE >= 1) AND (VALUE <= 12)));

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
    forms text[]
);