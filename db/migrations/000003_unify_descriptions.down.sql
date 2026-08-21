-- Reverse of 000003. Best-effort by nature:
--
--  * The DLC relabel is not reversible. `the-teal-mask` / `the-indigo-disk`
--    descriptions come back as `scarlet-violet`, because the migration
--    deliberately discarded the distinction.
--  * Regrouping games into version groups requires every game in a group to
--    agree on text. Where they disagree (only possible if data changed after
--    the up migration ran) the lowest-ordered game's text wins.
--  * pokemon.dex_entries is restored as a duplicate of the dexentries table,
--    matching the redundant state that existed before.

BEGIN;

CREATE TABLE public.dexentries (
    pokemon_id integer NOT NULL,
    game text NOT NULL,
    entry text NOT NULL
);

INSERT INTO public.dexentries (pokemon_id, game, entry)
SELECT pokemon_id, version::text, text
FROM public.pokemondescriptions
ORDER BY pokemon_id, version;

ALTER TABLE ONLY public.dexentries
    ADD CONSTRAINT dexentries_pokemon_id_fkey FOREIGN KEY (pokemon_id) REFERENCES public.pokemon(id);
CREATE INDEX idx_dexentries_pokemon_id ON public.dexentries USING btree (pokemon_id);

ALTER TABLE public.pokemon ADD COLUMN dex_entries jsonb[];
UPDATE public.pokemon p
SET dex_entries = COALESCE((
    SELECT array_agg(jsonb_build_object('game', d.version::text, 'entry', d.text) ORDER BY d.version)
    FROM public.pokemondescriptions d
    WHERE d.pokemon_id = p.id
), '{}'::jsonb[]);
ALTER TABLE public.pokemon ALTER COLUMN dex_entries SET NOT NULL;

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
    ('scarlet-violet','scarlet'), ('scarlet-violet','violet');

ALTER TABLE public.move    ADD COLUMN descriptions jsonb[];
ALTER TABLE public.ability ADD COLUMN descriptions jsonb[];

-- Collapse (entity, game, text) back to one element per distinct text, carrying
-- the set of version groups whose games all resolve to that text.
WITH per_vg AS (
    SELECT DISTINCT ON (d.move_id, g.vg)
           d.move_id AS id, g.vg, d.text
    FROM public.movedescriptions d
    JOIN vg_game g ON g.game = d.version
    ORDER BY d.move_id, g.vg, d.version
), grouped AS (
    SELECT id, text, array_agg(vg ORDER BY vg) AS vgs, min(vg) AS first_vg
    FROM per_vg GROUP BY id, text
)
UPDATE public.move m
SET descriptions = COALESCE((
    SELECT array_agg(jsonb_build_object('entry', gr.text, 'versionGroup', to_jsonb(gr.vgs)) ORDER BY gr.first_vg)
    FROM grouped gr WHERE gr.id = m.id
), '{}'::jsonb[]);

WITH per_vg AS (
    SELECT DISTINCT ON (d.ability_id, g.vg)
           d.ability_id AS id, g.vg, d.text
    FROM public.abilitydescriptions d
    JOIN vg_game g ON g.game = d.version
    ORDER BY d.ability_id, g.vg, d.version
), grouped AS (
    SELECT id, text, array_agg(vg ORDER BY vg) AS vgs, min(vg) AS first_vg
    FROM per_vg GROUP BY id, text
)
UPDATE public.ability a
SET descriptions = COALESCE((
    SELECT array_agg(jsonb_build_object('entry', gr.text, 'versionGroup', to_jsonb(gr.vgs)) ORDER BY gr.first_vg)
    FROM grouped gr WHERE gr.id = a.id
), '{}'::jsonb[]);

ALTER TABLE public.move    ALTER COLUMN descriptions SET NOT NULL;
ALTER TABLE public.ability ALTER COLUMN descriptions SET NOT NULL;

DROP TABLE public.pokemondescriptions;
DROP TABLE public.movedescriptions;
DROP TABLE public.abilitydescriptions;
-- vg_game is ON COMMIT DROP, but it still depends on public.game right now,
-- so it has to go before the type does.
DROP TABLE vg_game;
DROP TYPE public.game;

COMMIT;
