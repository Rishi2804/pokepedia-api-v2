-- name: GetSpeciesByID :one
SELECT id, name FROM species WHERE id = $1;

-- name: GetSpeciesByName :one
SELECT id, name FROM species WHERE name = $1;

-- name: GetSpeciesIdFromPokemon :one
SELECT species_id FROM pokemon WHERE id = $1;

-- name: GetPokemonBySpeciesID :many
SELECT id, species_id, name, gen, type1, type2, weight, height, gender_rate,
       hp, atk, def, spatk, spdef, speed, bst, forms
FROM pokemon
WHERE species_id = $1
ORDER BY id;