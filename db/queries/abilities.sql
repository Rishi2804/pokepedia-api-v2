-- name: GetAbilitiesList :many
SELECT id, name, gen 
FROM ability 
ORDER BY id, gen ASC;

-- name: GetAbility :one
SELECT id, name, gen, effect, descriptions::text[] AS descriptions
FROM ability 
WHERE id = $1;

-- name: GetAbilityByName :one
SELECT id, name, gen, effect, descriptions::text[] AS descriptions
FROM ability 
WHERE name = $1;

-- name: GetPokemonLearnableAbility :many
SELECT DISTINCT p.id, p.name, p.gen, p.type1, COALESCE(p.type2::text, '') AS type2, p.species_id 
FROM abilitydetails
JOIN pokemon p ON p.id = pokemon_id
WHERE ability_id = $1
ORDER BY p.species_id, p.id ASC;

