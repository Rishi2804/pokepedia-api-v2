-- Columns for a stored display name and a Showdown id on the older entity
-- tables, the way 000013 and 000016 gave them to item.
--
--   display_name   the name to show. Rebuilding it from the slug with
--                  util.FormatName gets "Mr. Mime", "U-turn", "King's Shield",
--                  "Porygon-Z" and "RKS System" wrong, and cannot name forms
--                  ("Mega Charizard X", "Alolan Raichu") at all.
--   showdown_id    Showdown's own id (toID of its name), so callers stop
--                  guessing it from the slug and keeping their own override
--                  tables.
--
-- pokemon also records what the battle engine does with a battle-only forme:
--   battle_only_base_id  the pokemon a team submits for it (Mega Charizard X
--                        -> Charizard)
--   required_item_id     the item that reaches it (Charizardite X)
--   gigantamax           true for the Gmax formes
-- and showdown_name, Showdown's name for the species ("Charizard-Mega-X"),
-- which pasted teams and exports use.
--
-- The columns are added empty and nullable here. 000024 fills them from the
-- scraper (cmd/scrape -only names), then makes display_name NOT NULL and adds
-- the UNIQUE and foreign-key constraints, which can only hold once every row
-- is filled. The same two-step as 000016's item.display_name.

BEGIN;

ALTER TABLE public.species ADD COLUMN display_name text;

ALTER TABLE public.pokemon
    ADD COLUMN display_name        text,
    ADD COLUMN showdown_id         text,
    ADD COLUMN showdown_name       text,
    ADD COLUMN battle_only_base_id integer,
    ADD COLUMN required_item_id    integer,
    ADD COLUMN gigantamax          boolean NOT NULL DEFAULT false;

ALTER TABLE public.move
    ADD COLUMN display_name text,
    ADD COLUMN showdown_id  text;

ALTER TABLE public.ability
    ADD COLUMN display_name text,
    ADD COLUMN showdown_id  text;

COMMIT;
