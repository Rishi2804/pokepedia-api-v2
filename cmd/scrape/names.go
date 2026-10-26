package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The Showdown dumps are pokepedia-battle's `npm run dumpdex` output. Joining
// them onto the database is what gives pokemon, move and ability rows the
// exact Showdown id the battle engine wants, in place of the name-guessing the
// UI and battle server each did with their own copies.
//
//go:embed data/showdown-species.json
var showdownSpeciesJSON []byte

//go:embed data/showdown-moves.json
var showdownMovesJSON []byte

//go:embed data/showdown-abilities.json
var showdownAbilitiesJSON []byte

type showdownSpecies struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	BaseSpecies      string   `json:"baseSpecies"`
	Forme            string   `json:"forme"`
	BattleOnlyBaseID *string  `json:"battleOnlyBaseId"`
	RequiredItemIDs  []string `json:"requiredItemIds"`
	Gigantamax       bool     `json:"gigantamax"`
}

type showdownNamed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type showdownDex struct {
	Species   map[string]showdownSpecies
	Moves     map[string]showdownNamed
	Abilities map[string]showdownNamed
}

func loadShowdownDex() (*showdownDex, error) {
	var species []showdownSpecies
	var moves, abilities []showdownNamed
	for _, d := range []struct {
		name string
		raw  []byte
		into any
	}{
		{"species", showdownSpeciesJSON, &species},
		{"moves", showdownMovesJSON, &moves},
		{"abilities", showdownAbilitiesJSON, &abilities},
	} {
		if err := json.Unmarshal(d.raw, d.into); err != nil {
			return nil, fmt.Errorf("decode showdown-%s.json: %w", d.name, err)
		}
	}
	dex := &showdownDex{
		Species:   make(map[string]showdownSpecies, len(species)),
		Moves:     make(map[string]showdownNamed, len(moves)),
		Abilities: make(map[string]showdownNamed, len(abilities)),
	}
	for _, s := range species {
		dex.Species[s.ID] = s
	}
	for _, m := range moves {
		dex.Moves[m.ID] = m
	}
	for _, a := range abilities {
		dex.Abilities[a.ID] = a
	}
	return dex, nil
}

// nameRow is one move or ability row: its stored display name and Showdown id.
type nameRow struct {
	ID          int32
	DisplayName string
	ShowdownID  *string
}

type speciesNameRow struct {
	ID          int32
	DisplayName string
}

type pokemonNameRow struct {
	ID               int32
	DisplayName      string
	ShowdownID       *string
	ShowdownName     *string
	BattleOnlyBaseID *int32 // pokemon.id of the species a team submits for this forme
	RequiredItemID   *int32 // item.id that reaches this forme (Mega Stone, Rusted Sword, ...)
	Gigantamax       bool
}

type nameData struct {
	Species   []speciesNameRow
	Pokemon   []pokemonNameRow
	Moves     []nameRow
	Abilities []nameRow
}

// showdownSpeciesID is the Showdown species id for one of our slugs.
func showdownSpeciesID(slug string) string {
	if id, ok := showdownSpeciesOverrides[slug]; ok {
		return id
	}
	return toShowdownID(slug)
}

// Showdown's regional formes ("Alola", "Galar-Zen") read as adjectives.
var regionalAdjectives = map[string]string{
	"Alola":  "Alolan",
	"Galar":  "Galarian",
	"Hisui":  "Hisuian",
	"Paldea": "Paldean",
}

// pokemonDisplayName builds the name shown for a pokemon row from its
// species' English name (PokeAPI's: "Mr. Mime", "Farfetch'd", "Nidoran♀")
// and Showdown's forme: "Mega Charizard X", "Alolan Raichu", "Gigantamax
// Urshifu (Rapid Strike)", "Necrozma (Dusk Mane)". The default forme is just
// the species name.
func pokemonDisplayName(speciesName, forme string) string {
	if forme == "" {
		return speciesName
	}
	rest := func(prefix string) string {
		return strings.TrimPrefix(strings.TrimPrefix(forme, prefix), "-")
	}
	paren := func(s string) string {
		if s == "" {
			return ""
		}
		words := strings.Split(s, "-")
		for i, w := range words {
			// Showdown abbreviates the gendered formes: Meowstic-F, Indeedee-F.
			switch w {
			case "F":
				words[i] = "Female"
			case "M":
				words[i] = "Male"
			}
		}
		return " (" + strings.Join(words, " ") + ")"
	}

	switch {
	case forme == "Mega" || strings.HasPrefix(forme, "Mega-"):
		// Mega-X / Mega-Y read as a suffix: "Mega Charizard X".
		suffix := rest("Mega")
		if suffix != "" {
			suffix = " " + suffix
		}
		return "Mega " + speciesName + suffix
	case strings.HasSuffix(forme, "-Mega"):
		// A Mega of a non-default forme: Meowstic-F-Mega, Tatsugiri-Droopy-Mega.
		return "Mega " + speciesName + paren(strings.TrimSuffix(forme, "-Mega"))
	case strings.HasSuffix(forme, "Gmax"):
		inner := strings.TrimSuffix(strings.TrimSuffix(forme, "Gmax"), "-")
		return "Gigantamax " + speciesName + paren(inner)
	case forme == "Primal":
		return "Primal " + speciesName
	}
	for region, adjective := range regionalAdjectives {
		if forme == region || strings.HasPrefix(forme, region+"-") {
			return adjective + " " + speciesName + paren(rest(region))
		}
	}
	return speciesName + paren(forme)
}

// fallbackPokemonName names a form Showdown has no species for (cosmetic
// forms such as Alcremie's flavors or Pikachu's caps): the species name plus
// whatever the slug adds after the species slug, title-cased.
func fallbackPokemonName(speciesName, speciesSlug, slug string) string {
	remainder := strings.TrimPrefix(strings.TrimPrefix(slug, speciesSlug), "-")
	if remainder == "" || remainder == slug {
		return speciesName
	}
	return speciesName + " (" + titleCaseWords(remainder) + ")"
}

func titleCaseWords(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// scrapeNames builds display names and Showdown ids for every species,
// pokemon, move and ability row, plus each forme's battle-only base and
// required item. Names come from PokeAPI's English names (already cached by
// the other passes) and the Showdown dump; anything that can't be matched is
// reported rather than guessed at, so a miss is visible instead of silent.
func scrapeNames(ctx context.Context, pool *pgxpool.Pool, cacheDir string) (*nameData, []unresolvedItem, error) {
	cache, err := newDiskCache(cacheDir + "/pokeapi")
	if err != nil {
		return nil, nil, err
	}
	dex, err := loadShowdownDex()
	if err != nil {
		return nil, nil, err
	}

	var unresolved []unresolvedItem
	data := &nameData{}

	// --- species: PokeAPI's English species name ---
	speciesRows, err := fetchEntities(ctx, pool, `SELECT id, name FROM species ORDER BY id`)
	if err != nil {
		return nil, nil, err
	}
	speciesName := map[int32]string{}
	speciesSlug := map[int32]string{}
	for _, s := range speciesRows {
		name, err := fetchPokeAPIName(cache, "pokemon-species", s.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("species %d (%s): %w", s.ID, s.Name, err)
		}
		name = titleCharReplacements.Replace(name)
		if name == "" {
			name = titleCaseWords(s.Name)
			unresolved = append(unresolved, unresolvedf("species %d (%s): no English PokeAPI name; used %q", s.ID, s.Name, name))
		}
		speciesName[s.ID] = name
		speciesSlug[s.ID] = s.Name
		data.Species = append(data.Species, speciesNameRow{ID: s.ID, DisplayName: name})
	}

	// --- pokemon: species name + Showdown forme, joined to the dump ---
	pokemonRows, err := pool.Query(ctx, `SELECT id, name, species_id FROM pokemon ORDER BY id`)
	if err != nil {
		return nil, nil, err
	}
	type pokemonRef struct {
		id        int32
		slug      string
		speciesID int32
	}
	var pokemon []pokemonRef
	for pokemonRows.Next() {
		var p pokemonRef
		if err := pokemonRows.Scan(&p.id, &p.slug, &p.speciesID); err != nil {
			pokemonRows.Close()
			return nil, nil, err
		}
		pokemon = append(pokemon, p)
	}
	pokemonRows.Close()
	if err := pokemonRows.Err(); err != nil {
		return nil, nil, err
	}

	itemRows, err := pool.Query(ctx, `SELECT id, showdown_id FROM item WHERE showdown_id IS NOT NULL`)
	if err != nil {
		return nil, nil, err
	}
	itemByShowdown := map[string]int32{}
	for itemRows.Next() {
		var id int32
		var sd string
		if err := itemRows.Scan(&id, &sd); err != nil {
			itemRows.Close()
			return nil, nil, err
		}
		itemByShowdown[sd] = id
	}
	itemRows.Close()
	if err := itemRows.Err(); err != nil {
		return nil, nil, err
	}

	pokemonByShowdown := map[string]int32{}
	rows := make([]pokemonNameRow, len(pokemon))
	showdownOf := make([]*showdownSpecies, len(pokemon))
	for i, p := range pokemon {
		base := speciesName[p.speciesID]
		row := pokemonNameRow{ID: p.id}
		sd, ok := dex.Species[showdownSpeciesID(p.slug)]
		if !ok {
			row.DisplayName = fallbackPokemonName(base, speciesSlug[p.speciesID], p.slug)
			unresolved = append(unresolved, unresolvedf("pokemon %d (%s): no Showdown species %q; named %q", p.id, p.slug, showdownSpeciesID(p.slug), row.DisplayName))
			rows[i] = row
			continue
		}
		if taken, dup := pokemonByShowdown[sd.ID]; dup {
			row.DisplayName = fallbackPokemonName(base, speciesSlug[p.speciesID], p.slug)
			unresolved = append(unresolved, unresolvedf("pokemon %d (%s): Showdown id %q already used by pokemon %d; left unmatched", p.id, p.slug, sd.ID, taken))
			rows[i] = row
			continue
		}
		pokemonByShowdown[sd.ID] = p.id
		sdID, sdName := sd.ID, sd.Name
		row.ShowdownID, row.ShowdownName = &sdID, &sdName
		row.DisplayName = pokemonDisplayName(base, sd.Forme)
		row.Gigantamax = sd.Gigantamax
		rows[i] = row
		showdownOf[i] = &sd
	}
	// Battle-only base and required item need every row's id first.
	for i := range rows {
		sd := showdownOf[i]
		if sd == nil {
			continue
		}
		if sd.BattleOnlyBaseID != nil {
			baseID, ok := pokemonByShowdown[*sd.BattleOnlyBaseID]
			if !ok {
				// Greninja-Ash's base is Greninja-Bond, which this database
				// doesn't have; teams submit the base species instead, the same
				// fallback the battle server's resolveBattleOnly makes.
				if base, known := dex.Species[*sd.BattleOnlyBaseID]; known {
					baseID, ok = pokemonByShowdown[toShowdownID(base.BaseSpecies)]
				}
			}
			if ok {
				rows[i].BattleOnlyBaseID = &baseID
			} else {
				unresolved = append(unresolved, unresolvedf("pokemon %d (%s): battle-only base %q is not in the database", rows[i].ID, sd.ID, *sd.BattleOnlyBaseID))
			}
		}
		for _, itemSD := range sd.RequiredItemIDs {
			if itemID, ok := itemByShowdown[itemSD]; ok {
				rows[i].RequiredItemID = &itemID
				break
			}
			unresolved = append(unresolved, unresolvedf("pokemon %d (%s): required item %q is not in the item table", rows[i].ID, sd.ID, itemSD))
		}
	}
	data.Pokemon = rows

	// --- moves and abilities ---
	moves, err := fetchEntities(ctx, pool, movesQuery)
	if err != nil {
		return nil, nil, err
	}
	abilities, err := fetchEntities(ctx, pool, abilitiesQuery)
	if err != nil {
		return nil, nil, err
	}
	if data.Moves, err = nameRows(cache, "move", moves, func(slug string) (showdownNamed, bool) {
		id, ok := showdownMoveOverrides[slug]
		if !ok {
			id = toShowdownID(slug)
		}
		sd, ok := dex.Moves[id]
		return sd, ok
	}, &unresolved); err != nil {
		return nil, nil, err
	}
	if data.Abilities, err = nameRows(cache, "ability", abilities, func(slug string) (showdownNamed, bool) {
		sd, ok := dex.Abilities[toShowdownID(slug)]
		return sd, ok
	}, &unresolved); err != nil {
		return nil, nil, err
	}

	sort.Slice(unresolved, func(i, j int) bool { return unresolved[i].Message < unresolved[j].Message })
	return data, unresolved, nil
}

// nameRows names moves or abilities from PokeAPI's English name and matches
// each to Showdown by normalised slug. Two rows landing on one Showdown id
// leave the later one unmatched, since showdown_id is unique.
func nameRows(
	cache *diskCache, kind string, items []dbEntity,
	lookup func(slug string) (showdownNamed, bool), unresolved *[]unresolvedItem,
) ([]nameRow, error) {
	used := map[string]int32{}
	out := make([]nameRow, 0, len(items))
	for _, it := range items {
		name, err := fetchPokeAPIName(cache, kind, it.ID)
		if err != nil {
			return nil, fmt.Errorf("%s %d (%s): %w", kind, it.ID, it.Name, err)
		}
		name = titleCharReplacements.Replace(name)
		if name == "" {
			name = titleCaseWords(it.Name)
			*unresolved = append(*unresolved, unresolvedf("%s %d (%s): no English PokeAPI name; used %q", kind, it.ID, it.Name, name))
		}
		row := nameRow{ID: it.ID, DisplayName: name}
		if sd, ok := lookup(it.Name); !ok {
			if reason, known := knownNameGaps[kind+":"+it.Name]; known {
				*unresolved = append(*unresolved, unresolvedItem{
					Message: fmt.Sprintf("%s %d (%s): no Showdown %s", kind, it.ID, it.Name, kind),
					Known:   true, Reason: reason,
				})
			} else {
				*unresolved = append(*unresolved, unresolvedf("%s %d (%s): no Showdown %s %q", kind, it.ID, it.Name, kind, toShowdownID(it.Name)))
			}
		} else if taken, dup := used[sd.ID]; dup {
			*unresolved = append(*unresolved, unresolvedf("%s %d (%s): Showdown id %q already used by %s %d; left unmatched", kind, it.ID, it.Name, sd.ID, kind, taken))
		} else {
			used[sd.ID] = it.ID
			id := sd.ID
			row.ShowdownID = &id
		}
		out = append(out, row)
	}
	return out, nil
}
