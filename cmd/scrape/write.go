package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// writeRows upserts every row. overwrite selects DO UPDATE instead of the
// default DO NOTHING (see queries.go) -- gap-fill-only is the default so
// that re-running the scraper after a successful pass is a true no-op:
// existing rows never change, only missing (id, version) pairs are added.
func writeRows(ctx context.Context, pool *pgxpool.Pool, rows []descriptionRow, overwrite bool) error {
	if len(rows) == 0 {
		return nil
	}

	conflict := "NOTHING"
	if overwrite {
		conflict = "UPDATE SET text = EXCLUDED.text"
	}

	queries := map[entity]string{
		entityPokemon: fmt.Sprintf(upsertPokemonDescription, conflict),
		entityMove:    fmt.Sprintf(upsertMoveDescription, conflict),
		entityAbility: fmt.Sprintf(upsertAbilityDescription, conflict),
	}

	batch := &pgx.Batch{}
	for _, r := range rows {
		q, ok := queries[r.Entity]
		if !ok {
			return fmt.Errorf("unknown entity %q for row id=%d version=%s", r.Entity, r.ID, r.Version)
		}
		batch.Queue(q, r.ID, r.Version, r.Text)
	}

	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	for range rows {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch exec: %w", err)
		}
	}
	return nil
}
