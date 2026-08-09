-- name: GetDexNational :many
SELECT s.id AS species_id, p.id, s.name, p.gen, p.type1, p.type2
FROM pokemon p
JOIN species s ON s.id = p.species_id
ORDER BY s.id, p.id ASC;

-- name: GetDexByRegion :many
SELECT d.num, p.id, s.name, s.id AS species_id, p.gen,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN pt.type2 ELSE p.type2 END AS type2
FROM dexnumber d
JOIN pokemon p ON d.default_variate = p.id OR d.alt_variates @> ARRAY[p.id]
JOIN species s ON s.id = d.species_id
LEFT JOIN pasttypes pt ON p.id = pt.pokemon_id AND pt.gen >= $1
WHERE d.name = $2
ORDER BY d.num, CASE WHEN p.id = d.default_variate THEN 0 ELSE 1 END, p.id;

-- name: GetTeamCandidatesNational :many
-- p.name (not s.name) so an alternate form's own name/slug is used, not its
-- species' base name — a national Alolan Raichu must read "raichu-alola",
-- not "raichu", the same way the regional queries below already do.
SELECT s.id AS num, s.id AS species_id, p.id, p.name, p.gen, p.type1, p.type2, p.gender_rate
FROM pokemon p
JOIN species s ON s.id = p.species_id
ORDER BY s.id, p.id ASC;

-- name: GetTeamCandidatesByRegion :many
SELECT d.num, p.id, p.name, p.gen, p.gender_rate,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN pt.type2 ELSE p.type2 END AS type2
FROM dexnumber d
JOIN pokemon p ON d.default_variate = p.id OR d.alt_variates @> ARRAY[p.id]
JOIN species s ON s.id = d.species_id
LEFT JOIN pasttypes pt ON p.id = pt.pokemon_id AND pt.gen >= $1
WHERE d.name = $2
ORDER BY d.num, CASE WHEN p.id = d.default_variate THEN 0 ELSE 1 END, p.id;

-- name: GetTeamCandidatesNationalByVersionGroup :many
SELECT DISTINCT p.id, p.name, p.gen, p.gender_rate, p.species_id,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN pt.type2 ELSE p.type2 END AS type2
FROM pokemon p
JOIN movedetails md ON p.id = md.pokemon_id
LEFT JOIN pasttypes pt ON p.id = pt.pokemon_id AND pt.gen >= $1
WHERE md.version = $2
ORDER BY p.species_id, p.id;

-- name: GetTeamCandidateByID :one
-- Single candidate for a non-national version group. The EXISTS pair is the same
-- membership union the list endpoint builds — in one of the version group's regional
-- dexes, or learning a move in that version — so a Pokemon absent from the game
-- yields no row. The lateral picks the earliest applicable past-typing record
-- deterministically, which a plain LEFT JOIN could not guarantee for a :one query.
SELECT p.id, p.name, p.gen, p.gender_rate,
       p.hp, p.atk, p.def, p.spatk, p.spdef, p.speed, p.bst,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN pt.type2 ELSE p.type2 END AS type2
FROM pokemon p
LEFT JOIN LATERAL (
    SELECT type1, type2
    FROM pasttypes
    WHERE pokemon_id = p.id AND gen >= sqlc.arg(gen)::public.generation
    ORDER BY gen
    LIMIT 1
) pt ON TRUE
WHERE p.id = sqlc.arg(pokemon_id)::int
  AND (
    EXISTS (
      SELECT 1 FROM dexnumber d
      WHERE d.name::text = ANY(sqlc.arg(regions)::text[])
        AND (d.default_variate = p.id OR d.alt_variates @> ARRAY[p.id])
    )
    OR EXISTS (
      SELECT 1 FROM movedetails md
      WHERE md.pokemon_id = p.id AND md.version = sqlc.arg(version)::public."group"
    )
  );

-- name: GetTeamCandidateNationalByID :one
-- Mirrors GetTeamCandidatesNational: the pokemon's own name (so an alternate
-- form's name/slug is used, not its species' base name), raw types, and no
-- membership check since the national list contains every Pokemon.
SELECT p.id, p.name, p.gen, p.type1, p.type2, p.gender_rate,
       p.hp, p.atk, p.def, p.spatk, p.spdef, p.speed, p.bst
FROM pokemon p
WHERE p.id = $1;

-- name: GetCandidateAbilitiesByIDs :many
SELECT d.pokemon_id, a.id AS ability_id, a.name AS ability_name, a.gen AS ability_gen,
       d.hidden, d.gen AS gen_removed
FROM abilitydetails d
JOIN ability a ON a.id = d.ability_id
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY d.pokemon_id, d.hidden, a.id;

-- name: GetCandidateMovesByIDsAndVersion :many
SELECT DISTINCT d.pokemon_id, d.move_id, m.name, m.type, m.class
FROM movedetails d
JOIN move m ON d.move_id = m.id
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[]) AND d.version = sqlc.arg(version)
ORDER BY d.pokemon_id, d.move_id;

-- name: GetCandidateMovesByIDs :many
SELECT DISTINCT d.pokemon_id, d.move_id, m.name, m.type, m.class
FROM movedetails d
JOIN move m ON d.move_id = m.id
WHERE d.pokemon_id = ANY(sqlc.arg(pokemon_ids)::int[])
ORDER BY d.pokemon_id, d.move_id;