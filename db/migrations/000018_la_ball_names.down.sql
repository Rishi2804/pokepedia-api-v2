-- Reverse of 000018.

BEGIN;

UPDATE public.item AS i SET display_name = v.display_name
FROM (VALUES
  ('lapoke-ball', 'Poké Ball'),
  ('lagreat-ball', 'Great Ball'),
  ('laultra-ball', 'Ultra Ball'),
  ('laheavy-ball', 'Heavy Ball'),
  ('lastrange-ball', 'Strange Ball'),
  ('lafeather-ball', 'Feather Ball'),
  ('lawing-ball', 'Wing Ball'),
  ('lajet-ball', 'Jet Ball'),
  ('laleaden-ball', 'Leaden Ball'),
  ('lagigaton-ball', 'Gigaton Ball'),
  ('laorigin-ball', 'Origin Ball')
) AS v(name, display_name)
WHERE i.name = v.name;

COMMIT;
