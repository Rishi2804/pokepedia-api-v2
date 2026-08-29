// Command scrape backfills missing rows in pokemondescriptions,
// movedescriptions, and abilitydescriptions from two sources:
//
//   - PokeAPI: Scarlet/Violet and Legends: Arceus move and ability
//     descriptions. PokeAPI already serves this text; this database's
//     original import simply never re-ran after those games shipped.
//
//   - Bulbapedia: every per-form Pokemon dex entry, for every game, plus
//     Legends: Z-A move descriptions. PokeAPI cannot serve either --
//     pokemon-form has no flavor_text_entries field, and pokemon-species
//     returns only the base form's text, so alt forms (Megas, regional
//     forms, ...) have no PokeAPI source at all; and Legends: Z-A has no
//     PokeAPI flavor text yet for any entity.
//
// -dry-run builds and reports counts without writing to Postgres. -emit also
// writes the built rows as a golang-migrate .up.sql file, so a run intended
// to be permanent survives a db/init.sql regeneration.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/config"
)

func main() {
	source := flag.String("source", "both", "pokeapi | bulbapedia | both")
	only := flag.String("only", "pokemon,move,ability", "comma-separated entity types to scrape")
	dryRun := flag.Bool("dry-run", false, "fetch and parse without writing to Postgres")
	emit := flag.String("emit", "", "also write a golang-migrate .up.sql file with the same rows")
	cacheDir := flag.String("cache", "cmd/scrape/.cache", "on-disk cache directory for scraped responses")
	batch := flag.Int("batch", 20, "Bulbapedia titles per MediaWiki request")
	limit := flag.Int("limit", 0, "cap the number of entities processed per type (0 = no cap)")
	overwrite := flag.Bool("overwrite", false, "DO UPDATE instead of DO NOTHING on existing (id, version) rows")
	verboseKnown := flag.Bool("verbose-known", false, "also print the full detail line for every known/diagnosed gap (see cmd/scrape/knowngaps.go)")
	oldGens := flag.Bool("old-gens", false, "with -only=eggmoves, also fetch older-generation breeding subpages (Generation II-VIII learnset); off by default so a first run only touches gen 9, which is already cached")
	flag.Parse()

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer pool.Close()

	types := map[string]bool{}
	for _, t := range strings.Split(*only, ",") {
		types[strings.TrimSpace(t)] = true
	}

	sources := map[string]bool{}
	if *source == "both" {
		sources["pokeapi"] = true
		sources["bulbapedia"] = true
	} else {
		sources[*source] = true
	}

	// eggmoves is a separate pipeline (movedetails, not *descriptions -- see
	// moveDetailRow in types.go) run instead of, not alongside, the
	// description passes: -only=eggmoves is meant to be used on its own.
	if types["eggmoves"] {
		if !sources["bulbapedia"] {
			log.Fatal("eggmoves requires -source=bulbapedia (or the default \"both\"); PokeAPI has no egg-move breeding data")
		}
		eggRows, unresolved, err := scrapeEggMoves(ctx, pool, *limit, *batch, *cacheDir, *oldGens)
		if err != nil {
			log.Fatalf("eggmoves pass failed: %v", err)
		}

		if *verboseKnown {
			reportEggMovesVerbose(eggRows, unresolved)
		} else {
			reportEggMoves(eggRows, unresolved)
		}

		if *dryRun {
			return
		}

		if err := writeEggMoveRows(ctx, pool, eggRows); err != nil {
			log.Fatalf("write failed: %v", err)
		}
		fmt.Printf("wrote %d egg-move rows to Postgres\n", len(eggRows))

		if *emit != "" {
			if err := emitEggMoveMigration(*emit, eggRows); err != nil {
				log.Fatalf("emit failed: %v", err)
			}
			fmt.Printf("wrote migration: %s\n", *emit)
		}
		return
	}

	var rows []descriptionRow
	var unresolved []unresolvedItem

	if sources["pokeapi"] {
		r, err := scrapePokeAPI(ctx, pool, types, *limit, *cacheDir)
		if err != nil {
			log.Fatalf("pokeapi pass failed: %v", err)
		}
		rows = append(rows, r...)
	}

	if sources["bulbapedia"] {
		r, u, err := scrapeBulbapedia(ctx, pool, types, *limit, *batch, *cacheDir)
		if err != nil {
			log.Fatalf("bulbapedia pass failed: %v", err)
		}
		rows = append(rows, r...)
		unresolved = append(unresolved, u...)
	}

	if *verboseKnown {
		reportVerbose(rows, unresolved)
	} else {
		report(rows, unresolved)
	}

	if *dryRun {
		return
	}

	if err := writeRows(ctx, pool, rows, *overwrite); err != nil {
		log.Fatalf("write failed: %v", err)
	}
	fmt.Printf("wrote %d rows to Postgres\n", len(rows))

	if *emit != "" {
		if err := emitMigration(*emit, rows); err != nil {
			log.Fatalf("emit failed: %v", err)
		}
		fmt.Printf("wrote migration: %s\n", *emit)
	}
}
