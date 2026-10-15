-- Prefix the Legends: Arceus balls with "LA ".
--
-- PokeAPI names five of them exactly like the mainline balls, so the item list
-- showed "Poké Ball", "Great Ball", "Ultra Ball", "Heavy Ball" and "Strange
-- Ball" twice each with no way to tell which was which. All eleven get the
-- prefix so the family stays together; the mainline balls keep their names.
-- Re-run `make index` afterwards so search picks up the new names.

BEGIN;

UPDATE public.item AS i SET display_name = v.display_name
FROM (VALUES
  ('lapoke-ball', 'LA Poké Ball'),
  ('lagreat-ball', 'LA Great Ball'),
  ('laultra-ball', 'LA Ultra Ball'),
  ('laheavy-ball', 'LA Heavy Ball'),
  ('lastrange-ball', 'LA Strange Ball'),
  ('lafeather-ball', 'LA Feather Ball'),
  ('lawing-ball', 'LA Wing Ball'),
  ('lajet-ball', 'LA Jet Ball'),
  ('laleaden-ball', 'LA Leaden Ball'),
  ('lagigaton-ball', 'LA Gigaton Ball'),
  ('laorigin-ball', 'LA Origin Ball')
) AS v(name, display_name)
WHERE i.name = v.name;

COMMIT;
