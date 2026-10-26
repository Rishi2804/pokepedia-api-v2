-- name: GetItemsList :many
SELECT id, name, display_name, gen, category, pocket, image
FROM item
ORDER BY id;

-- name: GetItem :one
SELECT * FROM item WHERE id = $1;

-- name: GetItemByName :one
SELECT * FROM item WHERE name = $1;

-- name: GetItemDescriptions :many
-- Same grouping as GetAbilityDescriptions; see the note on
-- GetPokemonDescriptionsByIDs.
SELECT array_agg(version ORDER BY version)::text[] AS games,
       text
FROM itemdescriptions
WHERE item_id = $1
GROUP BY text
ORDER BY min(version);

-- name: GetPokemonHoldingItem :many
SELECT DISTINCT p.id, p.name, p.display_name, p.gen, p.type1, p.type2, p.species_id
FROM itemheldby h
JOIN pokemon p ON p.id = h.pokemon_id
WHERE h.item_id = $1
ORDER BY p.species_id, p.id ASC;

-- name: GetHoldableItems :many
-- The team builder's item picker: items that are held in battle AND that the
-- battle server accepts in the given format ('2'..'9', 'nationaldex',
-- 'legendsza'). legal_formats comes from Showdown's own dex, so a pick here
-- can't fail the server's team validation.
SELECT showdown_id, display_name, image, short_effect
FROM item
WHERE holdable
  AND showdown_id IS NOT NULL
  AND $1::text = ANY(legal_formats)
ORDER BY display_name;
