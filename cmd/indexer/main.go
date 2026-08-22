// Command indexer builds documents for the pokepedia-search Elasticsearch
// index from Postgres. -dry-run builds and validates them without writing
// anything; the bulk-index + alias-swap path is added separately.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Rishi2804/pokepedia-api-v2/internal/config"
)

func main() {
	only := flag.String("only", "pokemon,move,ability", "comma-separated entity types to build")
	dryRun := flag.Bool("dry-run", false, "build documents and report counts without writing to Elasticsearch")
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

	docs, err := buildDocs(ctx, pool, types)
	if err != nil {
		log.Fatalf("failed to build documents: %v", err)
	}

	counts := map[string]int{}
	for _, d := range docs {
		counts[d.Type]++
	}
	var typeOrder []string
	for t := range counts {
		typeOrder = append(typeOrder, t)
	}
	sort.Strings(typeOrder)

	fmt.Printf("built %d documents:\n", len(docs))
	for _, t := range typeOrder {
		fmt.Printf("  %-8s %d\n", t, counts[t])
	}

	if *dryRun {
		return
	}

	log.Fatal("bulk indexing not implemented yet — run with -dry-run")
}

func buildDocs(ctx context.Context, pool *pgxpool.Pool, types map[string]bool) ([]indexDoc, error) {
	species, err := fetchSpecies(ctx, pool)
	if err != nil {
		return nil, err
	}
	speciesNames := map[int32]string{}
	for _, s := range species {
		if s.Name != nil {
			speciesNames[s.ID] = *s.Name
		}
	}

	var docs []indexDoc

	if types["pokemon"] {
		rows, err := fetchPokemon(ctx, pool)
		if err != nil {
			return nil, err
		}
		descriptions, err := fetchDescriptions(ctx, pool, pokemonDescriptionsQuery)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			docs = append(docs, buildPokemonDoc(r, speciesNames[r.SpeciesID], descriptions))
		}
	}

	if types["move"] {
		rows, err := fetchMoves(ctx, pool)
		if err != nil {
			return nil, err
		}
		descriptions, err := fetchDescriptions(ctx, pool, moveDescriptionsQuery)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			docs = append(docs, buildMoveDoc(r, descriptions))
		}
	}

	if types["ability"] {
		rows, err := fetchAbilities(ctx, pool)
		if err != nil {
			return nil, err
		}
		descriptions, err := fetchDescriptions(ctx, pool, abilityDescriptionsQuery)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			docs = append(docs, buildAbilityDoc(r, descriptions))
		}
	}

	return docs, nil
}
