-- name: GetPokemon :one
SELECT id, species_id, name, gen, type1, COALESCE(type2::text, '') AS type2, weight, height, gender_rate,
       hp, atk, def, spatk, spdef, speed, bst, forms
FROM pokemon
WHERE id = $1;

-- name: GetPokemonByName :one
SELECT id, species_id, name, gen, type1, COALESCE(type2::text, '') AS type2, weight, height, gender_rate,
       hp, atk, def, spatk, spdef, speed, bst, forms
FROM pokemon
WHERE name = $1;

-- name: GetDexNumbers :many
SELECT species_id, name, num, default_variate, alt_variates
FROM dexnumber
WHERE default_variate = $1 OR alt_variates @> ARRAY[$1::int];

-- name: GetDexEntries :many
SELECT pokemon_id, game, entry
FROM dexentries
WHERE pokemon_id = $1;

-- name: GetEvolutionChain :many
SELECT chain_id, id, from_pokemon, from_display, to_pokemon, to_display,
       details, region, alt_form, next_evo, prev_evo
FROM get_evolution_chain_by_id($1);

-- name: GetPokemonMoves :many
SELECT
    d.move_id, m.name, m.type, d.level_learned, d.method,
    COALESCE(d.version::text, '') AS version, m.class,
    COALESCE(pmv.power, m.power) AS power,
    COALESCE(pmv.accuracy, m.accuracy) AS accuracy,
    COALESCE(pmv.pp, m.pp) AS pp
FROM movedetails d
JOIN move m ON d.move_id = m.id
LEFT JOIN pastmovevalues pmv ON d.move_id = pmv.id AND pmv.version_groups @> ARRAY[d.version]
WHERE d.pokemon_id = $1;

-- name: GetPokemonAbilities :many
SELECT a.id AS ability_id, a.name AS ability_name, a.gen AS ability_gen,
       d.hidden, COALESCE(d.gen::int, 0) AS gen_removed
FROM abilitydetails d
JOIN ability a ON a.id = d.ability_id
WHERE d.pokemon_id = $1;