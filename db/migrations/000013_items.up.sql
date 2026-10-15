-- Item tables, following the ability / abilitydescriptions pattern.
--
-- `item.name` is the PokeAPI slug (e.g. `choice-scarf`). Items that exist in
-- Showdown or on Bulbapedia but not in PokeAPI (the Legends: Z-A Mega Stones)
-- get ids from 20000 up so they can never collide with a future PokeAPI id.
--
-- `showdown_id` is the Showdown item id (`choicescarf`) that
-- PokemonTeamMember.item and the battle server use; it is NULL for items
-- Showdown does not model (Potion, Repel, ...).
--
-- `holdable` + `legal_formats` drive the team builder's item picker. An item is
-- holdable only if it is both holdable in the games and known to Showdown;
-- `legal_formats` holds the BattleFormatKey values ('2'..'9', 'nationaldex',
-- 'legendsza') the battle server will accept it in. The data comes from
-- pokepedia-battle's `npm run dumpitems`, so the picker and the server's
-- validation cannot disagree.
--
-- TMs/TRs/HMs (pocket `machines`) are deliberately not stored: ~400
-- near-duplicate rows that move pages already cover.

BEGIN;

CREATE TABLE public.item (
    id            integer     PRIMARY KEY,
    name          text        NOT NULL,
    category      text        NOT NULL,
    pocket        text        NOT NULL,
    gen           public.generation,
    cost          integer     NOT NULL DEFAULT 0,
    fling_power   integer,
    fling_effect  text,
    effect        text,
    short_effect  text,
    sprite        text,
    showdown_id   text        UNIQUE,
    holdable      boolean     NOT NULL DEFAULT false,
    legal_formats text[]      NOT NULL DEFAULT '{}'
);

CREATE TABLE public.itemdescriptions (
    item_id integer     NOT NULL REFERENCES public.item(id),
    version public.game NOT NULL,
    text    text        NOT NULL,
    PRIMARY KEY (item_id, version)
);

-- Wild held items (PokeAPI `held_by_pokemon`), one row per game with its
-- encounter-slot rarity percentage.
CREATE TABLE public.itemheldby (
    item_id    integer     NOT NULL REFERENCES public.item(id),
    pokemon_id integer     NOT NULL REFERENCES public.pokemon(id),
    version    public.game NOT NULL,
    rarity     integer     NOT NULL,
    PRIMARY KEY (item_id, pokemon_id, version)
);

CREATE INDEX idx_item_name           ON public.item (name);
CREATE INDEX idx_item_name_trgm      ON public.item USING gin (name gin_trgm_ops);
CREATE INDEX idx_item_showdown_id    ON public.item (showdown_id);
CREATE INDEX idx_itemheldby_pokemon  ON public.itemheldby (pokemon_id);

COMMIT;
