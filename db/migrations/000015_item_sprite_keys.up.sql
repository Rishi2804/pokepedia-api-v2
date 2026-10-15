-- Give every item its conventional image key.
--
-- 000014 only set `sprite` for items PokeAPI ships an image for. The key is
-- really a naming convention -- items/<slug>.png in the pokemon-images bucket,
-- filled by object_store/fetch-item-images.sh -- and that script also pulls
-- bag images from Bulbapedia for holdable items PokeAPI has none for (the
-- Legends: Z-A Mega Stones, among others). Setting the key everywhere keeps
-- the API simple; the UI shows a placeholder where no image exists.

BEGIN;

UPDATE public.item SET sprite = 'items/' || name || '.png' WHERE sprite IS NULL;

COMMIT;
