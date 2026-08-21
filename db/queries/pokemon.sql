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
SELECT
    d.pokemon_id, d.move_id, m.name, m.type, d.level_learned, d.method,
    d.version, m.class,
    COALESCE(pmv.power, m.power) AS power,
    COALESCE(pmv.accuracy, m.accuracy) AS accuracy,
    COALESCE(pmv.pp, m.pp) AS pp
FROM movedetails d
JOIN move m ON d.move_id = m.id
LEFT JOIN pastmovevalues pmv ON d.move_id = pmv.id AND pmv.version_groups @> ARRAY[d.version]
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY d.pokemon_id, d.version, d.method, d.level_learned, d.move_id;

-- name: GetPokemonAbilitiesByIDs :many
SELECT d.pokemon_id, a.id AS ability_id, a.name AS ability_name, a.gen AS ability_gen,
       d.hidden, d.gen AS gen_removed
FROM abilitydetails d
JOIN ability a ON a.id = d.ability_id
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY d.pokemon_id, d.hidden, a.id;