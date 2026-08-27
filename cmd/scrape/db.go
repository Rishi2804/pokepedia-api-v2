package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// dbEntity is a row's id and its DB slug (e.g. "thunderbolt", "clefable-mega"),
// used both to build outbound requests (PokeAPI ids are literal DB ids;
// Bulbapedia titles are derived from the entity's name) and, for pokemon, to
// build the FormatName reverse index in resolve.go.
type dbEntity struct {
	ID   int32
	Name string
}

func fetchEntities(ctx context.Context, pool *pgxpool.Pool, query string) ([]dbEntity, error) {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dbEntity
	for rows.Next() {
		var e dbEntity
		if err := rows.Scan(&e.ID, &e.Name); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func capEntities(items []dbEntity, limit int) []dbEntity {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}
