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

// movedetails has no primary key or unique constraint -- 8,575
// (pokemon_id, move_id, method, version) groups already have duplicates,
// all of them method='level-up' and all legitimate (the same move learned
// at two different levels across sub-versions, e.g. Bulbasaur's Vine Whip
// at both level 7 and 9 in Sun/Moon). Egg moves have zero such duplicates
// today, so a WHERE NOT EXISTS guard gives the same gap-fill-only
// idempotency the ON CONFLICT upserts above give the *descriptions tables,
// without a schema change.
const insertEggMove = `
INSERT INTO public.movedetails (pokemon_id, move_id, method, level_learned, version)
SELECT $1, $2, 'egg', 0, $3
WHERE NOT EXISTS (
  SELECT 1 FROM public.movedetails
  WHERE pokemon_id = $1 AND move_id = $2 AND method = 'egg' AND version = $3
)
`

// insertLegendsMove is insertEggMove's Legends: Arceus/Z-A counterpart:
// movedetails still has no unique constraint, so this stays a
// WHERE NOT EXISTS guard rather than ON CONFLICT. level_learned is part of
// the guard (unlike insertEggMove, where it's always 0) so a genuine
// two-different-levels case, if one ever exists, isn't blocked by an
// unrelated row at a different level for the same move.
const insertLegendsMove = `
INSERT INTO public.movedetails (pokemon_id, move_id, method, level_learned, version)
SELECT $1, $2, $3, $4, $5
WHERE NOT EXISTS (
  SELECT 1 FROM public.movedetails
  WHERE pokemon_id = $1 AND move_id = $2 AND method = $3 AND level_learned = $4 AND version = $5
)
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
