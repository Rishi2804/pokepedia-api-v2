package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// versionGroupGames maps a PokeAPI version_group slug to the public.game
// enum value(s) whose flavor text is identical within that group -- true for
// every paired-version generation this database already stores (e.g.
// sword-shield's rows are duplicated onto both "sword" and "shield"). Only
// the two version groups this database's original import stopped short of
// are listed: scarlet-violet and legends-arceus. Every earlier group is
// already fully populated, so re-scraping them would be wasted requests.
var versionGroupGames = map[string][]string{
	"scarlet-violet": {"scarlet", "violet"},
	"legends-arceus": {"legends-arceus"},
}

type pokeAPIFlavorText struct {
	FlavorText   string                `json:"flavor_text"`
	Language     struct{ Name string } `json:"language"`
	VersionGroup struct{ Name string } `json:"version_group"`
}

type pokeAPIEntityResp struct {
	FlavorTextEntries []pokeAPIFlavorText `json:"flavor_text_entries"`
	Names             []struct {
		Name     string                `json:"name"`
		Language struct{ Name string } `json:"language"`
	} `json:"names"`
}

// scrapePokeAPI backfills Scarlet/Violet and Legends: Arceus move and ability
// descriptions. PokeAPI serves this text today; the gap is that this
// database's original import never re-ran after Scarlet/Violet shipped, not
// that the data is unavailable -- verified live (pound, thunderbolt, outrage,
// night-slash all carry scarlet-violet and legends-arceus flavor_text_entries).
//
// Pokemon are deliberately absent from this pass: PokeAPI's per-form
// endpoint (pokemon-form) carries no flavor_text_entries field at all, and
// pokemon-species returns only the base form's text, so it cannot serve the
// alt-form entries this project actually needs. All Pokemon description
// scraping lives in bulba.go instead.
func scrapePokeAPI(ctx context.Context, pool *pgxpool.Pool, types map[string]bool, limit int, cacheDir string) ([]descriptionRow, error) {
	cache, err := newDiskCache(cacheDir + "/pokeapi")
	if err != nil {
		return nil, err
	}

	var rows []descriptionRow

	if types["move"] {
		r, err := scrapePokeAPIEntity(ctx, pool, cache, movesQuery, "move", entityMove, limit)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}

	if types["ability"] {
		r, err := scrapePokeAPIEntity(ctx, pool, cache, abilitiesQuery, "ability", entityAbility, limit)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}

	return rows, nil
}

func scrapePokeAPIEntity(
	ctx context.Context, pool *pgxpool.Pool, cache *diskCache,
	idQuery, pokeAPIKind string, ent entity, limit int,
) ([]descriptionRow, error) {
	items, err := fetchEntities(ctx, pool, idQuery)
	if err != nil {
		return nil, err
	}
	items = capEntities(items, limit)

	var rows []descriptionRow
	for _, item := range items {
		resp, err := fetchPokeAPIEntity(cache, pokeAPIKind, item.ID)
		if err != nil {
			return nil, fmt.Errorf("%s %d (%s): %w", pokeAPIKind, item.ID, item.Name, err)
		}
		for _, e := range resp.FlavorTextEntries {
			if e.Language.Name != "en" {
				continue
			}
			games, ok := versionGroupGames[e.VersionGroup.Name]
			if !ok {
				continue
			}
			text := cleanFlavorText(e.FlavorText)
			for _, g := range games {
				rows = append(rows, descriptionRow{
					Entity: ent, ID: item.ID, Version: g, Text: text, Source: "pokeapi",
				})
			}
		}
	}
	return rows, nil
}

// fetchPokeAPIRaw fetches and caches https://pokeapi.co/api/v2/<kind>/<id>/
// verbatim. Both the flavor-text pass above and the Bulbapedia title
// resolution in resolve.go read from the same cached response for "move"
// and "pokemon-species" ids -- one network fetch serves both purposes
// instead of each half of the tool re-fetching the same URL under a
// different cache key.
//
// Always fetched by numeric id, never by name: DB ids are verbatim PokeAPI
// ids (this whole database was seeded from PokeAPI), so this is exact and
// sidesteps any slug drift or URL-encoding concern that fetching by name
// would introduce.
func fetchPokeAPIRaw(cache *diskCache, kind string, id int32) ([]byte, error) {
	key := fmt.Sprintf("%s-%d.json", kind, id)
	if body, ok := cache.get(key); ok {
		return body, nil
	}
	url := fmt.Sprintf("https://pokeapi.co/api/v2/%s/%d/", kind, id)
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	if err := cache.put(key, body); err != nil {
		return nil, err
	}
	return body, nil
}

func fetchPokeAPIEntity(cache *diskCache, kind string, id int32) (pokeAPIEntityResp, error) {
	body, err := fetchPokeAPIRaw(cache, kind, id)
	if err != nil {
		return pokeAPIEntityResp{}, err
	}
	var r pokeAPIEntityResp
	if err := json.Unmarshal(body, &r); err != nil {
		return pokeAPIEntityResp{}, fmt.Errorf("decode %s-%d: %w", kind, id, err)
	}
	return r, nil
}

// cleanFlavorText collapses PokeAPI's embedded newline/form-feed line breaks
// (inserted to wrap text for the in-game textbox) into single spaces, and
// trims trailing whitespace some existing rows in this database already
// carry from the original import but which new rows should not repeat.
func cleanFlavorText(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\f", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
