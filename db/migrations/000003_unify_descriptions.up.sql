-- Unify description storage.
--
-- Before this migration descriptive text lived in four shapes: the `dexentries`
-- relation table (per game, untyped text column, no PK), a fully redundant
-- `pokemon.dex_entries jsonb[]` copy that no query ever read, and
-- `move.descriptions` / `ability.descriptions` as `jsonb[]` of
-- {"entry": ..., "versionGroup": [...]} keyed per version-group set.
--
-- After it there is one pattern: (<entity>_id, version, text) with a real
-- primary key, a foreign key, and an enum-typed version column. Grouping
-- identical text across games happens at read time in SQL.

BEGIN;

-- Release order is load-bearing: Postgres sorts enums by declaration order, so
-- `ORDER BY version` yields release order natively and the hand-maintained
-- pokeenum.GameOrder / GameIndex sort in Go becomes unnecessary.
--
-- No DLC members. The Paldea DLC is the only case where PokeAPI minted separate
-- version groups (`the-teal-mask`, `the-indigo-disk`); Isle of Armor and Crown
-- Tundra were folded into `sword-shield` upstream and exist only as Pokedexes.
-- We follow the majority convention: DLC is a Pokemon-data dimension (the
-- `dex` enum, which already carries isle-of-armor/crown-tundra/kitakami/
-- blueberry), never a description dimension. The two DLC slugs are relabelled
-- to scarlet-violet in the backfill below.
CREATE TYPE public.game AS ENUM (
    'red','blue','yellow','gold','silver','crystal','ruby','sapphire','emerald',
    'firered','leafgreen','diamond','pearl','platinum','heartgold','soulsilver',
    'black','white','black-2','white-2','x','y','omega-ruby','alpha-sapphire',
    'sun','moon','ultra-sun','ultra-moon','lets-go-pikachu','lets-go-eevee',
    'sword','shield','brilliant-diamond','shining-pearl','legends-arceus',
    'scarlet','violet'
);

CREATE TABLE public.pokemondescriptions (
    pokemon_id integer     NOT NULL REFERENCES public.pokemon(id),
    version    public.game NOT NULL,
    text       text        NOT NULL,
    PRIMARY KEY (pokemon_id, version)
);

CREATE TABLE public.movedescriptions (
    move_id integer     NOT NULL REFERENCES public.move(id),
    version public.game NOT NULL,
    text    text        NOT NULL,
    PRIMARY KEY (move_id, version)
);

CREATE TABLE public.abilitydescriptions (
    ability_id integer     NOT NULL REFERENCES public.ability(id),
    version    public.game NOT NULL,
    text       text        NOT NULL,
    PRIMARY KEY (ability_id, version)
);

-- Version-group -> games. 21 real version groups plus the two DLC slugs folded
-- onto scarlet/violet. Only needed here; once descriptions are game-grained
-- nothing maps version groups at request time.
CREATE TEMP TABLE vg_game (vg text NOT NULL, game public.game NOT NULL) ON COMMIT DROP;
INSERT INTO vg_game (vg, game) VALUES
    ('red-blue','red'), ('red-blue','blue'),
    ('yellow','yellow'),
    ('gold-silver','gold'), ('gold-silver','silver'),
    ('crystal','crystal'),
    ('ruby-sapphire','ruby'), ('ruby-sapphire','sapphire'),
    ('emerald','emerald'),
    ('firered-leafgreen','firered'), ('firered-leafgreen','leafgreen'),
    ('diamond-pearl','diamond'), ('diamond-pearl','pearl'),
    ('platinum','platinum'),
    ('heartgold-soulsilver','heartgold'), ('heartgold-soulsilver','soulsilver'),
    ('black-white','black'), ('black-white','white'),
    ('black-2-white-2','black-2'), ('black-2-white-2','white-2'),
    ('x-y','x'), ('x-y','y'),
    ('omega-ruby-alpha-sapphire','omega-ruby'), ('omega-ruby-alpha-sapphire','alpha-sapphire'),
    ('sun-moon','sun'), ('sun-moon','moon'),
    ('ultra-sun-ultra-moon','ultra-sun'), ('ultra-sun-ultra-moon','ultra-moon'),
    ('lets-go-pikachu-lets-go-eevee','lets-go-pikachu'), ('lets-go-pikachu-lets-go-eevee','lets-go-eevee'),
    ('sword-shield','sword'), ('sword-shield','shield'),
    ('brilliant-diamond-and-shining-pearl','brilliant-diamond'), ('brilliant-diamond-and-shining-pearl','shining-pearl'),
    ('legends-arceus','legends-arceus'),
    ('scarlet-violet','scarlet'), ('scarlet-violet','violet'),
    -- DLC relabelled onto their base games rather than given enum members.
    ('the-teal-mask','scarlet'),   ('the-teal-mask','violet'),
    ('the-indigo-disk','scarlet'), ('the-indigo-disk','violet');

-- Pokedex flavor: straight copy. (pokemon_id, game) is already unique.
INSERT INTO public.pokemondescriptions (pokemon_id, version, text)
SELECT pokemon_id, game::public.game, entry
FROM public.dexentries;

-- Move / ability descriptions: explode the jsonb[] into (entity, game, text).
--
-- Tie-break, which matters in two places: a natively-scarlet-violet entry must
-- beat a relabelled DLC one (Queenly Majesty's the-teal-mask entry is Gulp
-- Missile's text by a source-scrape misalignment, and it sits AFTER the good
-- scarlet-violet row, so a plain `ord DESC` would select the corrupt text);
-- among entries of equal provenance the later array element wins (Insomnia has
-- two competing sword-shield texts).
INSERT INTO public.movedescriptions (move_id, version, text)
SELECT DISTINCT ON (t.id, t.game) t.id, t.game, t.entry
FROM (
    SELECT m.id,
           u.ord,
           u.elem->>'entry' AS entry,
           vg.slug          AS vg,
           g.game
    FROM public.move m,
         unnest(m.descriptions) WITH ORDINALITY AS u(elem, ord),
         LATERAL jsonb_array_elements_text(u.elem->'versionGroup') AS vg(slug)
         JOIN vg_game g ON g.vg = vg.slug
) t
ORDER BY t.id, t.game, (t.vg IN ('the-teal-mask','the-indigo-disk')), t.ord DESC;

INSERT INTO public.abilitydescriptions (ability_id, version, text)
SELECT DISTINCT ON (t.id, t.game) t.id, t.game, t.entry
FROM (
    SELECT a.id,
           u.ord,
           u.elem->>'entry' AS entry,
           vg.slug          AS vg,
           g.game
    FROM public.ability a,
         unnest(a.descriptions) WITH ORDINALITY AS u(elem, ord),
         LATERAL jsonb_array_elements_text(u.elem->'versionGroup') AS vg(slug)
         JOIN vg_game g ON g.vg = vg.slug
) t
ORDER BY t.id, t.game, (t.vg IN ('the-teal-mask','the-indigo-disk')), t.ord DESC;

-- Old shapes.
DROP INDEX IF EXISTS public.idx_dexentries_pokemon_id;
DROP TABLE public.dexentries;
ALTER TABLE public.pokemon  DROP COLUMN dex_entries;
ALTER TABLE public.move     DROP COLUMN descriptions;
ALTER TABLE public.ability  DROP COLUMN descriptions;

-- Primary keys that only ever existed in db/init.sql, never in the migration
-- path, so a migrate-built database drifted from a Docker-seeded one. Checked
-- by conrelid rather than by name: the seeded ability PK is `ablility_pkey`.
DO $$
DECLARE
    t text;
BEGIN
    -- Normalise the historical typo before the existence check below.
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ablility_pkey') THEN
        ALTER TABLE public.ability RENAME CONSTRAINT ablility_pkey TO ability_pkey;
    END IF;

    -- All five are keyed on (id), matching the definitions in db/init.sql.
    FOREACH t IN ARRAY ARRAY['ability','move','pokemon','species','evolution'] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE contype = 'p' AND conrelid = format('public.%I', t)::regclass
        ) THEN
            EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I PRIMARY KEY (id)', t, t || '_pkey');
        END IF;
    END LOOP;
END $$;

COMMIT;
