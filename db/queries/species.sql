-- name: GetSpeciesByID :one
SELECT id, name FROM species WHERE id = $1;

-- name: GetSpeciesByName :one
SELECT id, name FROM species WHERE name = $1;

-- name: GetPokemonSummariesBySpeciesID :many
SELECT id, name, gen, type1, COALESCE(type2::text, '') AS type2
FROM pokemon
WHERE species_id = $1
ORDER BY id;