-- name: GetDexNational :many
SELECT s.id AS species_id, p.id, s.name, p.gen, p.type1, COALESCE(p.type2::text, '') AS type2
FROM pokemon p
JOIN species s ON s.id = p.species_id
ORDER BY s.id, p.id ASC;

-- name: GetDexByRegion :many
SELECT d.num, p.id, s.name, s.id AS species_id, p.gen,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN COALESCE(pt.type2::text, '') ELSE COALESCE(p.type2::text, '') END AS type2
FROM dexnumber d
JOIN pokemon p ON d.default_variate = p.id OR d.alt_variates @> ARRAY[p.id]
JOIN species s ON s.id = d.species_id
LEFT JOIN pasttypes pt ON p.id = pt.pokemon_id AND pt.gen >= $1
WHERE d.name = $2
ORDER BY d.num, CASE WHEN p.id = d.default_variate THEN 0 ELSE 1 END;

-- name: GetTeamCandidatesNational :many
SELECT s.id AS num, s.id AS species_id, p.id, s.name, p.gen, p.type1, COALESCE(p.type2::text, '') AS type2
FROM pokemon p
JOIN species s ON s.id = p.species_id
ORDER BY s.id, p.id ASC;

-- name: GetTeamCandidatesByRegion :many
SELECT d.num, p.id, p.name, p.gen, p.gender_rate,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN COALESCE(pt.type2::text, '') ELSE COALESCE(p.type2::text, '') END AS type2
FROM dexnumber d
JOIN pokemon p ON d.default_variate = p.id OR d.alt_variates @> ARRAY[p.id]
JOIN species s ON s.id = d.species_id
LEFT JOIN pasttypes pt ON p.id = pt.pokemon_id AND pt.gen >= $1
WHERE d.name = $2
ORDER BY d.num, CASE WHEN p.id = d.default_variate THEN 0 ELSE 1 END;

-- name: GetTeamCandidatesNationalByVersionGroup :many
SELECT DISTINCT p.id, p.name, p.gen, p.gender_rate, p.species_id,
       COALESCE(pt.type1, p.type1) AS type1,
       CASE WHEN pt.type1 IS NOT NULL THEN COALESCE(pt.type2::text, '') ELSE COALESCE(p.type2::text, '') END AS type2
FROM pokemon p
JOIN movedetails md ON p.id = md.pokemon_id
LEFT JOIN pasttypes pt ON p.id = pt.pokemon_id AND pt.gen >= $1
WHERE md.version = $2
ORDER BY p.species_id, p.id;