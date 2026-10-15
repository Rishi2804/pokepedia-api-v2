-- Rename item.sprite to item.image.
--
-- The column holds the object-store key of the item's picture
-- (items/<slug>.png), and not all of those are sprites any more: most are
-- Bulbapedia bag and Dream World images at 120-160px, with PokeAPI's 30px
-- pixel sprites only as a fallback. "image" is the accurate name; the API
-- field, search index and UI follow.
--
-- Deploy together with the new API build: the old build selects `sprite` and
-- will fail against this schema. Re-run `make index` afterwards, since the
-- search documents carry the key as meta.image now.

BEGIN;

ALTER TABLE public.item RENAME COLUMN sprite TO image;

COMMIT;
