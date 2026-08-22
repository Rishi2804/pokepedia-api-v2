package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pokemonRow struct {
	ID         int32
	Slug       string
	Gen        int32
	Type1      string
	Type2      *string
	Weight     float64
	Height     float64
	HP         int32
	Atk        int32
	Def        int32
	SpAtk      int32
	SpDef      int32
	Speed      int32
	BST        int32
	SpeciesID  int32
	Popularity int32
}

type moveRow struct {
	ID       int32
	Slug     string
	Gen      int32
	Type     string
	Class    string
	Power    *int32
	Accuracy *int32
	PP       *int32
	Effect   *string
}

type abilityRow struct {
	ID     int32
	Slug   string
	Gen    int32
	Effect *string
}

type speciesRow struct {
	ID   int32
	Name *string
}

func fetchPokemon(ctx context.Context, pool *pgxpool.Pool) ([]pokemonRow, error) {
	rows, err := pool.Query(ctx, pokemonQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []pokemonRow
	for rows.Next() {
		var r pokemonRow
		if err := rows.Scan(&r.ID, &r.Slug, &r.Gen, &r.Type1, &r.Type2, &r.Weight, &r.Height,
			&r.HP, &r.Atk, &r.Def, &r.SpAtk, &r.SpDef, &r.Speed, &r.BST, &r.SpeciesID, &r.Popularity); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func fetchMoves(ctx context.Context, pool *pgxpool.Pool) ([]moveRow, error) {
	rows, err := pool.Query(ctx, moveQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []moveRow
	for rows.Next() {
		var r moveRow
		if err := rows.Scan(&r.ID, &r.Slug, &r.Gen, &r.Type, &r.Class, &r.Power, &r.Accuracy, &r.PP, &r.Effect); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func fetchAbilities(ctx context.Context, pool *pgxpool.Pool) ([]abilityRow, error) {
	rows, err := pool.Query(ctx, abilityQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []abilityRow
	for rows.Next() {
		var r abilityRow
		if err := rows.Scan(&r.ID, &r.Slug, &r.Gen, &r.Effect); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func fetchSpecies(ctx context.Context, pool *pgxpool.Pool) ([]speciesRow, error) {
	rows, err := pool.Query(ctx, speciesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []speciesRow
	for rows.Next() {
		var r speciesRow
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// fetchDescriptions maps entity id -> deduped description texts. An entity
// with zero description rows is simply absent from the map.
func fetchDescriptions(ctx context.Context, pool *pgxpool.Pool, query string) (map[int32][]string, error) {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int32][]string{}
	for rows.Next() {
		var id int32
		var texts []string
		if err := rows.Scan(&id, &texts); err != nil {
			return nil, err
		}
		out[id] = texts
	}
	return out, rows.Err()
}
