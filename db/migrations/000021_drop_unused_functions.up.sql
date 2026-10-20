-- Drop two SQL functions nothing calls.
--
-- Both came in with the baseline dump and were superseded by queries in
-- db/queries: get_pokemon_moves() by GetPokemonMovesByIDs (which also joins
-- legendsmovevalues, and has been the only moveset path since 000010), and
-- get_evolution_chain_by_name() by looking the Pokemon up by name first and
-- calling get_evolution_chain_by_id(). Neither appears in any query, service
-- or script. get_pokemon_moves() also hard-codes pastmovevalues.id, so it
-- would have broken at runtime when that column is renamed in 000022.
--
-- get_evolution_chain_by_id() stays: GetEvolutionChainByIDs calls it.
--
-- 000006's down is a no-op, so nothing else needs these functions back.

BEGIN;

DROP FUNCTION IF EXISTS public.get_pokemon_moves(integer);
DROP FUNCTION IF EXISTS public.get_evolution_chain_by_name(text);

COMMIT;
