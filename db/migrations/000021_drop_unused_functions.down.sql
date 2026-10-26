-- Reverse of 000021: recreates both functions as the baseline defined them.

BEGIN;

CREATE FUNCTION public.get_evolution_chain_by_name(pokemon_name text) RETURNS SETOF public.evolution
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY
    WITH pokemon_id AS (
        SELECT p.id
        FROM pokemon p
        WHERE p.name = pokemon_name
    ),
    chain_id_from_evolution AS (
        SELECT e.chain_id
        FROM evolution e
        WHERE e.from_pokemon IN (SELECT id FROM pokemon_id)
           OR e.to_pokemon IN (SELECT id FROM pokemon_id)
        LIMIT 1
    ),
    chain_id_from_evolutionpeek AS (
        SELECT ep.chain_id
        FROM evolutionpeek ep
        WHERE ep.pokemon_id IN (SELECT id FROM pokemon_id)
        LIMIT 1
    )

    SELECT e.*
    FROM evolution e
    WHERE e.chain_id IN (
        SELECT chain_id FROM chain_id_from_evolution
        UNION
        SELECT chain_id FROM chain_id_from_evolutionpeek
    );
END;
$$;

CREATE FUNCTION public.get_pokemon_moves(p_pokemon_id integer) RETURNS TABLE(move_id integer, name text, type public.ptype, level_learned integer, method public.learnmethod, version public."group", class public.mclass, power integer, accuracy integer, pp integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
  RETURN QUERY
  SELECT
    d.move_id,
    m.name,
    m.type,
    d.level_learned,
    d.method,
    d.version,
    m.class,
    CASE
      WHEN pmd.id IS NOT NULL AND pmd.version_groups @> ARRAY[d.version] THEN pmd.power
      ELSE m.power
    END AS power,
    CASE
      WHEN pmd.id IS NOT NULL AND pmd.version_groups @> ARRAY[d.version] THEN pmd.accuracy
      ELSE m.accuracy
    END AS accuracy,
    CASE
      WHEN pmd.id IS NOT NULL AND pmd.version_groups @> ARRAY[d.version] THEN pmd.pp
      ELSE m.pp
    END AS pp
  FROM movedetails d
  JOIN move m ON d.move_id = m.id
  LEFT JOIN pastmovevalues pmd ON d.move_id = pmd.id AND pmd.version_groups @> ARRAY[d.version]
  WHERE d.pokemon_id = p_pokemon_id;
END; $$;

COMMIT;
