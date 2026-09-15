-- Per-(pokemon, move, game) move stat overrides for Legends: Arceus and
-- Legends: Z-A. Both games shape move data differently from mainline:
--
--   Legends: Arceus  -- a "mastery" level, plus THREE power values (base /
--                        strong style / agile style) and two accuracy values
--   Legends: Z-A     -- a "plus" level and a cooldown (CD) instead of PP
--
-- This cannot reuse public.pastmovevalues: that table is keyed by move id
-- plus a version_groups array, so one row applies uniformly to every
-- Pokemon that knows the move. Verified against Bulbapedia's own data that
-- this is too coarse -- of 176 distinct Legends: Arceus moves, all but four
-- have identical stats across every Pokemon that learns them, and three of
-- those four are genuine per-FORM differences: Dialga/Palkia/Giratina's
-- Origin Formes have boosted signature moves (Roar of Time 120 base power
-- for dialga, 140 for dialga-origin; Shadow Force 100 vs 120; Spacial Rend
-- 90 vs 80 -- Palkia's numbers are reversed, its Origin Forme value is
-- LOWER). A pastmovevalues-shaped table would silently flatten these to one
-- value. Legends: Z-A had zero such inconsistencies across 339 moves, but
-- the same table serves both since the key is already per-Pokemon.
--
-- Not stored on public.movedetails itself: that table has no primary key
-- (8,575 legitimate duplicate level-up rows already exist -- see
-- cmd/scrape/queries.go's insertEggMove comment), so it is a poor place to
-- hang additional per-row attributes, and a move learnable two ways in one
-- game (Legends: Arceus's Iron Tail is both tutor and level 37 for Pikachu)
-- has identical stats regardless of learn method -- one row here serves
-- every movedetails row for that (pokemon, move, version) triple.
--
-- Columns are nullable and simply unused per game: Legends: Z-A rows leave
-- power_strong/power_agile/accuracy_2/pp null and populate cooldown;
-- Legends: Arceus rows do the reverse.

BEGIN;

CREATE TABLE public.legendsmovevalues (
    pokemon_id   integer NOT NULL REFERENCES public.pokemon(id),
    move_id      integer NOT NULL REFERENCES public.move(id),
    version      public."group" NOT NULL,
    second_level integer,   -- Legends: Arceus mastery level / Legends: Z-A "plus" level
    power_base   integer,
    power_strong integer,   -- Legends: Arceus only
    power_agile  integer,   -- Legends: Arceus only
    accuracy_1   integer,
    accuracy_2   integer,   -- Legends: Arceus only
    pp           integer,   -- Legends: Arceus only
    cooldown     integer,   -- Legends: Z-A only
    PRIMARY KEY (pokemon_id, move_id, version)
);

CREATE INDEX idx_legendsmovevalues_move_id ON public.legendsmovevalues (move_id);

COMMIT;
