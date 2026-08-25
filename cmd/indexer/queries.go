package main

// popularity: a species' pokemon row whose slug equals the species name is
// its default form (10); every other row sharing that species_id is a
// variant (4) — Mega/Gmax/regional forms always have such a sibling. 28
// species (Deoxys, Giratina, Aegislash, Minior, ...) have no plain-named
// form at all, only named ones (deoxys-normal, giratina-altered, ...); for
// those every form stays at 10 rather than arbitrarily demoting one.
//
// dexnumber.default_variate/alt_variates was tried first and rejected: it's
// scoped per regional dex, so a form can be "default" in one region and a
// variant nationally (Alolan Raichu is default_variate for the Alola dex,
// which wrongly scored it 10 via an EXISTS-across-any-dex check).
const pokemonQuery = `
SELECT p.id, p.name, p.gen::int, p.type1::text, p.type2::text, p.species_id,
       CASE
         WHEN NOT EXISTS (
           SELECT 1 FROM pokemon base
           WHERE base.species_id = p.species_id AND base.name = s.name
         ) THEN 10
         WHEN p.name = s.name THEN 10
         ELSE 4
       END AS popularity
FROM pokemon p
JOIN species s ON s.id = p.species_id
ORDER BY p.id
`

const moveQuery = `
SELECT id, name, gen::int, type::text, class::text, power, accuracy, pp, effect
FROM move
ORDER BY id
`

const abilityQuery = `
SELECT id, name, gen::int, effect
FROM ability
ORDER BY id
`

const speciesQuery = `
SELECT id, name
FROM species
ORDER BY id
`

// Deduped per entity: flavor text repeats verbatim across games, and
// indexing the same sentence 8x would inflate its term frequency.
const pokemonDescriptionsQuery = `
SELECT pokemon_id, array_agg(DISTINCT text) AS texts
FROM pokemondescriptions
GROUP BY pokemon_id
`

const moveDescriptionsQuery = `
SELECT move_id, array_agg(DISTINCT text) AS texts
FROM movedescriptions
GROUP BY move_id
`

const abilityDescriptionsQuery = `
SELECT ability_id, array_agg(DISTINCT text) AS texts
FROM abilitydescriptions
GROUP BY ability_id
`
