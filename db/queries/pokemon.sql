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

-- name: ListPokemon :many
SELECT id, species_id, name, gen, type1, type2, weight, height, gender_rate,
       hp, atk, def, spatk, spdef, speed, bst, forms
FROM pokemon
ORDER BY id
LIMIT 50;