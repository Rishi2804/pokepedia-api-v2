-- name: GetMoves :many
SELECT id, name, type, class, power, accuracy, pp, gen 
FROM move 
ORDER BY id, gen ASC;

-- name: GetMove :one
SELECT id, name, type, class, power, accuracy, pp, gen, effect
FROM move 
WHERE id = $1;

-- name: GetMoveByName :one
SELECT id, name, type, class, power, accuracy, pp, gen, effect
FROM move 
WHERE name = $1;

-- name: GetMoveDescriptions :many
-- Same grouping as GetPokemonDescriptionsByIDs; see the note there.
SELECT array_agg(version ORDER BY version)::text[] AS games,
       text
FROM movedescriptions
WHERE move_id = $1
GROUP BY text
ORDER BY min(version);

-- name: GetPokemonLearnableMove :many
SELECT DISTINCT p.species_id, p.id, p.name, p.type1, type2, md.method
FROM movedetails md
JOIN pokemon p ON p.id = md.pokemon_id
WHERE md.move_id = $1
ORDER BY p.species_id, p.id ASC;

-- name: GetPastMoveValues :many
SELECT id, power, accuracy, pp, version_groups
FROM pastmovevalues
WHERE id = $1;