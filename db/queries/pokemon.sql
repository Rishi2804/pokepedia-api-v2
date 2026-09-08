-- name: GetPokemon :one
SELECT id, species_id, name, gen, type1, type2, weight, height, gender_rate,
       hp, atk, def, spatk, spdef, speed, bst, forms
FROM pokemon
WHERE id = $1;

-- name: GetPokemonByName :one
SELECT id, species_id, name, gen, type1, type2, weight, height, gender_rate,
       hp, atk, def, spatk, spdef, speed, bst, forms
FROM pokemon
WHERE name = $1;

-- name: GetDexNumbersByIDs :many
SELECT ids.pokemon_id, d.species_id, d.name, d.num, d.default_variate, d.alt_variates
FROM unnest(sqlc.arg(pokemon_ids)::int[]) AS ids(pokemon_id)
JOIN dexnumber d ON d.default_variate = ids.pokemon_id OR d.alt_variates @> ARRAY[ids.pokemon_id]
ORDER BY ids.pokemon_id, d.name;

-- name: GetPokemonDescriptionsByIDs :many
-- One row per game. `version` is the public.game enum, declared in release
-- order, so ORDER BY sorts chronologically with no Go-side sort needed.
SELECT pokemon_id, version, text
FROM pokemondescriptions
WHERE pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY pokemon_id, version;

-- name: GetEvolutionChainByIDs :many
SELECT ids.pokemon_id, e.chain_id, e.id, e.from_pokemon, e.from_display, e.to_pokemon,
       e.to_display, e.details, e.region, e.alt_form, e.next_evo, e.prev_evo
FROM unnest(sqlc.arg(pokemon_ids)::int[]) AS ids(pokemon_id)
CROSS JOIN LATERAL get_evolution_chain_by_id(ids.pokemon_id) e
ORDER BY ids.pokemon_id, e.id;

-- name: GetPokemonMovesByIDs :many
-- legendsmovevalues holds Legends: Arceus/Z-A's per-(pokemon, move, game)
-- stats -- pastmovevalues can't: it's keyed by move only, and Dialga/Palkia/
-- Giratina's Origin Formes have genuinely different power for their own
-- signature moves in Legends: Arceus (see 000010's migration comment).
-- When a legendsmovevalues row exists it is authoritative for power/
-- accuracy/pp -- never silently patched from pastmovevalues/move, since a
-- Legends: Z-A row's NULL pp (that game has no PP stat; it has cooldown
-- instead) must stay NULL, not fall back to an unrelated mainline value.
SELECT
    d.pokemon_id, d.move_id, m.name, m.type, d.level_learned, d.method,
    d.version, m.class,
    CASE WHEN lmv.pokemon_id IS NOT NULL THEN lmv.power_base
         ELSE COALESCE(pmv.power, m.power) END AS power,
    CASE WHEN lmv.pokemon_id IS NOT NULL THEN lmv.accuracy_1
         ELSE COALESCE(pmv.accuracy, m.accuracy) END AS accuracy,
    CASE WHEN lmv.pokemon_id IS NOT NULL THEN lmv.pp
         ELSE COALESCE(pmv.pp, m.pp) END AS pp,
    lmv.second_level, lmv.power_strong, lmv.power_agile, lmv.accuracy_2, lmv.cooldown
FROM movedetails d
JOIN move m ON d.move_id = m.id
LEFT JOIN pastmovevalues pmv ON d.move_id = pmv.id AND pmv.version_groups @> ARRAY[d.version]
LEFT JOIN legendsmovevalues lmv ON lmv.pokemon_id = d.pokemon_id AND lmv.move_id = d.move_id AND lmv.version = d.version
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY d.pokemon_id, d.version, d.method, d.level_learned, d.move_id;

-- name: GetPokemonAbilitiesByIDs :many
SELECT d.pokemon_id, a.id AS ability_id, a.name AS ability_name, a.gen AS ability_gen,
       d.hidden, d.gen AS gen_removed
FROM abilitydetails d
JOIN ability a ON a.id = d.ability_id
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY d.pokemon_id, d.hidden, a.id;