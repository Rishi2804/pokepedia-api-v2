-- name: GetSpeciesByID :one
SELECT id, name FROM species WHERE id = $1;

-- name: GetSpeciesByName :one
SELECT id, name FROM species WHERE name = $1;

-- name: GetSpeciesIdFromPokemon :one
SELECT species_id FROM pokemon WHERE id = $1;

-- name: GetPokemonIdsBySpeciesID :many
SELECT id FROM pokemon WHERE species_id = $1 ORDER BY id ASC;