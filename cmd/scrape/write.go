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

// writeEggMoveRows inserts egg-move facts using insertEggMove's
// WHERE NOT EXISTS guard (see queries.go for why this can't be an
// ON CONFLICT upsert like writeRows above). There is no -overwrite
// equivalent here: gap-fill-only is the only mode, since deleting or
// replacing an existing movedetails row risks the reverse-mismatch cases
// (Politoed has 15 egg moves in this DB vs 5 on Bulbapedia) the plan
// explicitly chose never to touch.
func writeEggMoveRows(ctx context.Context, pool *pgxpool.Pool, rows []moveDetailRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, r := range rows {
		batch.Queue(insertEggMove, r.PokemonID, r.MoveID, r.Version)
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

// writeLegendsMoveRows writes both movedetails and legendsmovevalues for
// each row, using insertLegendsMove's WHERE NOT EXISTS guard and
// upsertLegendsMoveValues's ON CONFLICT DO NOTHING (see queries.go). There
// is no -overwrite equivalent, matching writeEggMoveRows above: both tables
// start at zero rows for these two games, so gap-fill-only is a no-op
// distinction on a first run and the safe default on any later one.
func writeLegendsMoveRows(ctx context.Context, pool *pgxpool.Pool, rows []legendsMoveRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, r := range rows {
		batch.Queue(insertLegendsMove, r.PokemonID, r.MoveID, r.Method, r.Level, r.Version)
		batch.Queue(upsertLegendsMoveValues, r.PokemonID, r.MoveID, r.Version,
			r.SecondLevel, r.PowerBase, r.PowerStrong, r.PowerAgile,
			r.Accuracy1, r.Accuracy2, r.PP, r.Cooldown)
	}

	br := pool.SendBatch(ctx, batch)
	defer br.Close()

	for range rows {
		if _, err := br.Exec(); err != nil { // insertLegendsMove
			return fmt.Errorf("batch exec: %w", err)
		}
		if _, err := br.Exec(); err != nil { // upsertLegendsMoveValues
			return fmt.Errorf("batch exec: %w", err)
		}
	}
	return nil
}
