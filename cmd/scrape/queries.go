package main

// Read queries. id/name pairs, not just ids: name is what pokeapi.go uses to
// build request URLs (it equals the PokeAPI slug, since these rows were
// originally imported from PokeAPI) and what resolve.go uses to build the
// FormatName reverse index for Bulbapedia's {{Dex/Form|...}} markers.
const movesQuery = `SELECT id, name FROM move ORDER BY id`
const abilitiesQuery = `SELECT id, name FROM ability ORDER BY id`
const pokemonQuery = `SELECT id, name FROM pokemon ORDER BY id`

// Write queries. All three description tables share the same
// (entity_id, version, text) shape with a composite primary key, so the
// upsert is naturally idempotent. DO NOTHING is the default -- see
// write.go -- because the 720 existing Scarlet/Violet rows and 216 existing
// alt-form rows came from an earlier, already-reviewed Bulbapedia pass and
// must not be silently overwritten by a re-run.
const upsertPokemonDescription = `
INSERT INTO public.pokemondescriptions (pokemon_id, version, text)
VALUES ($1, $2, $3)
ON CONFLICT (pokemon_id, version) DO %s
`

const upsertMoveDescription = `
INSERT INTO public.movedescriptions (move_id, version, text)
VALUES ($1, $2, $3)
ON CONFLICT (move_id, version) DO %s
`

const upsertAbilityDescription = `
INSERT INTO public.abilitydescriptions (ability_id, version, text)
VALUES ($1, $2, $3)
ON CONFLICT (ability_id, version) DO %s
`

// movedetails has a primary key (pokemon_id, move_id, method, level_learned,
// version) since 000020, so egg and Legends rows are ordinary gap-fill
// upserts: ON CONFLICT DO NOTHING, the same policy every other write path in
// this tool uses. Before that key existed these were WHERE NOT EXISTS guards,
// because 8,575 (pokemon_id, move_id, method, version) groups legitimately
// repeat at different levels (Bulbasaur's Vine Whip at 7 and 9 in Sun/Moon);
// level_learned is part of the key, so that case is still never blocked.
const insertEggMove = `
INSERT INTO public.movedetails (pokemon_id, move_id, method, level_learned, version)
VALUES ($1, $2, 'egg', 0, $3)
ON CONFLICT (pokemon_id, move_id, method, level_learned, version) DO NOTHING
`

// insertLegendsMove is insertEggMove's Legends: Arceus/Z-A counterpart.
const insertLegendsMove = `
INSERT INTO public.movedetails (pokemon_id, move_id, method, level_learned, version)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (pokemon_id, move_id, method, level_learned, version) DO NOTHING
`

// upsertLegendsMoveValues writes the per-(pokemon, move, game) stat extras.
// legendsmovevalues has a real primary key (unlike movedetails), so this is
// a plain ON CONFLICT DO NOTHING: gap-fill only, matching every other write
// path in this tool. A move learnable two ways in one game (Legends:
// Arceus's Iron Tail is both tutor and level 37 for Pikachu) writes this
// same row twice with identical values; the second write is a no-op.
const upsertLegendsMoveValues = `
INSERT INTO public.legendsmovevalues
  (pokemon_id, move_id, version, second_level, power_base, power_strong, power_agile, accuracy_1, accuracy_2, pp, cooldown)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (pokemon_id, move_id, version) DO NOTHING
`

// Item writes. item and itemdescriptions have real primary keys, so these are
// the same ON CONFLICT upserts the *descriptions tables use: DO NOTHING by
// default (a re-run is a no-op), DO UPDATE under -overwrite.
const upsertItem = `
INSERT INTO public.item
  (id, name, category, pocket, gen, cost, fling_power, fling_effect, effect, short_effect,
   image, showdown_id, holdable, legal_formats, display_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (id) DO %s
`

const itemOverwriteSet = `UPDATE SET name = EXCLUDED.name, category = EXCLUDED.category, pocket = EXCLUDED.pocket,
  gen = EXCLUDED.gen, cost = EXCLUDED.cost, fling_power = EXCLUDED.fling_power,
  fling_effect = EXCLUDED.fling_effect, effect = EXCLUDED.effect, short_effect = EXCLUDED.short_effect,
  image = EXCLUDED.image, showdown_id = EXCLUDED.showdown_id, holdable = EXCLUDED.holdable,
  legal_formats = EXCLUDED.legal_formats, display_name = EXCLUDED.display_name`

const upsertItemDescription = `
INSERT INTO public.itemdescriptions (item_id, version, text)
VALUES ($1, $2, $3)
ON CONFLICT (item_id, version) DO %s
`

const insertItemHeldBy = `
INSERT INTO public.itemheldby (item_id, pokemon_id, version, rarity)
VALUES ($1, $2, $3, $4)
ON CONFLICT (item_id, pokemon_id, version) DO NOTHING
`
