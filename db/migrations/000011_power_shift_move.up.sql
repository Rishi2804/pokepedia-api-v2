-- Adds the single move row missing from public.move that Legends: Arceus's
-- own learnset data needs: "Power Shift" (PokeAPI id 829, Generation VIII).
--
-- Discovered while building cmd/scrape's -only=legends pass (see
-- cmd/scrape/legends.go): 23 species' Legends: Arceus tutor movesets on
-- Bulbapedia teach Power Shift, and every one of those rows failed to
-- resolve with "unknown move" since id 829 was never imported. Checked
-- whether this was symptomatic of a wider gap: public.move's only other
-- missing ids in this range are 896-900 ("Torque" moves, Revavroom's
-- Generation IX/Teal Mask signature-move family) -- unrelated to either
-- Legends game and left alone as a separate, pre-existing gap out of scope
-- here.
--
-- effect is NULL because PokeAPI itself has no English effect_entries text
-- for this move (verified live) -- not a gap introduced by this migration;
-- 103 of the 832 existing move rows already have a NULL effect for the
-- same reason.

BEGIN;

INSERT INTO public.move (id, name, gen, type, class, power, accuracy, pp, effect)
VALUES (829, 'power-shift', 8, 'normal', 'status', NULL, NULL, 10, NULL)
ON CONFLICT (id) DO NOTHING;

COMMIT;
