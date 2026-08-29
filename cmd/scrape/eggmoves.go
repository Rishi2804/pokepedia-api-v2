package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// genBreedingConfig describes how one Bulbapedia generation's
// "By {{pkmn|breeding}}" section maps onto public.group version groups.
// Verified against Bulbasaur's actual egg-move counts in every generation
// (17/17 version groups matched PokeAPI exactly before any of this ran):
//
//	II    6 entries, no markers        -> gold-silver, crystal (imprecise: DB
//	                                       actually splits 6/5; one page can't
//	                                       express that, so Crystal may gain
//	                                       one extra row -- accepted, gap-fill
//	                                       only never deletes)
//	III   8 entries, no markers        -> ruby-sapphire, emerald, firered-leafgreen
//	IV   14 entries, 2 trailing "HGSS" -> diamond-pearl+platinum+heartgold-soulsilver
//	                                       by default; a trailing "HGSS" token
//	                                       REPLACES that with heartgold-soulsilver
//	                                       alone for just that entry (verified:
//	                                       12 unmarked + 2 HGSS-marked = DP 12 /
//	                                       Pt 12 / HGSS 14)
//	V    14 entries, no markers        -> black-white, black-2-white-2
//	VI   15 entries, no markers        -> x-y, omega-ruby-alpha-sapphire
//	VII  15 entries, no markers        -> sun-moon, ultra-sun-ultra-moon
//	VIII 18 entries, {{gameabbrev8|..}} block markers -> sword-shield (6) /
//	                                       brilliant-diamond-and-shining-pearl (12)
//	IX (main page) 4 entries, no markers -> scarlet-violet
//
// The template name itself encodes the generation ({{learnlist/breed4|...}},
// {{learnlist/breed8|...}}, ...), which is what TemplateSuffix selects.
type genBreedingConfig struct {
	TemplateSuffix string
	DefaultGroups  []string
	EntryOverride  map[string]string // trailing per-entry CODE -> single group replacing DefaultGroups for that entry
}

var genConfigs = map[string]genBreedingConfig{
	"II":  {TemplateSuffix: "2", DefaultGroups: []string{"gold-silver", "crystal"}},
	"III": {TemplateSuffix: "3", DefaultGroups: []string{"ruby-sapphire", "emerald", "firered-leafgreen"}},
	"IV": {TemplateSuffix: "4", DefaultGroups: []string{"diamond-pearl", "platinum", "heartgold-soulsilver"},
		EntryOverride: map[string]string{"HGSS": "heartgold-soulsilver"}},
	"V":    {TemplateSuffix: "5", DefaultGroups: []string{"black-white", "black-2-white-2"}},
	"VI":   {TemplateSuffix: "6", DefaultGroups: []string{"x-y", "omega-ruby-alpha-sapphire"}},
	"VII":  {TemplateSuffix: "7", DefaultGroups: []string{"sun-moon", "ultra-sun-ultra-moon"}},
	"VIII": {TemplateSuffix: "8"}, // no default: every entry must be under a block marker
	"IX":   {TemplateSuffix: "9", DefaultGroups: []string{"scarlet-violet"}},
}

// breedingFormOverrides maps a breeding-section form heading straight to
// the pokemon slug it resolves to, for headings describing a game-mechanics
// distinction resolveFormMarker's token-overlap heuristic has no way to
// reason about:
//
//   - Darmanitan's breeding page groups by REGION only ("Darmanitan",
//     "Galarian Darmanitan"), never distinguishing Standard from Zen Mode --
//     because Zen Mode is a battle-only transformation that never hatches
//     from an egg, so both headings correctly mean the standard (breedable)
//     form only, not either mode specifically.
//   - Greninja's "Battle Bond Greninja" heading is NOT Ash-Greninja
//     (greninja-ash, a battle-only transformed state with no breeding
//     compatibility of its own): verified live, its five breeding entries
//     are identical to plain "Greninja"'s, just each requiring a held Mirror
//     Herb to copy instead of direct compatibility. It's the same
//     "greninja" row described a second time, not a different one.
var breedingFormOverrides = map[knownGapKey]string{
	{555, "Darmanitan"}:           "darmanitan-standard",
	{555, "Galarian Darmanitan"}:  "darmanitan-galar-standard",
	{658, "Battle Bond Greninja"}: "greninja",
}

// gameAbbrevCodeToGroup maps a standalone {{gameabbrevN|CODE}} block marker
// (as opposed to the nested {{gameabbrevN|CODE}} inside a movedescentry --
// see gameAbbrevCodes in wikitext.go) to the public.group it restricts
// subsequent breeding entries to until the next marker. Verified live: gen
// VIII's breeding section splits into an SwSh block (6 entries) and a BDSP
// block (12). SV/ZA are defensive: no species observed carries a marker in
// its gen-9 breeding section at all (Legends: Z-A has no breeding
// mechanic), but an unrecognized marker is reported, not silently dropped.
var gameAbbrevCodeToGroup = map[string]string{
	"SwSh": "sword-shield",
	"BDSP": "brilliant-diamond-and-shining-pearl",
	"SV":   "scarlet-violet",
	"ZA":   "legends-za",
}

// findBreedingHeading locates "==...==By {{pkmn|breeding}}==...==" at
// whatever level it's actually written at -- verified inconsistent even
// across subpages of the same species (Bulbasaur's Generation VI subpage
// uses four "=", its Generation VII subpage uses five). Go's regexp engine
// (RE2) has no backreferences, so matching open/close at a shared variable
// level isn't a single regex; trying each fixed level in turn is.
func findBreedingHeading(wikitext string) (level, end int, found bool) {
	for l := 3; l <= 6; l++ {
		eq := strings.Repeat("=", l)
		re := regexp.MustCompile(`(?m)^` + eq + `\s*By \{\{pkmn\|breeding\}\}\s*` + eq + `\s*$`)
		if loc := re.FindStringIndex(wikitext); loc != nil {
			return l, loc[1], true
		}
	}
	return 0, 0, false
}

// breedingSection extracts the breeding section body and the heading level
// it was found at (needed by findFormHeadings to look for sub-headings one
// level deeper), bounded by the next heading at or above that level.
func breedingSection(wikitext string) (body string, level int, found bool) {
	level, end, found := findBreedingHeading(wikitext)
	if !found {
		return "", 0, false
	}
	rest := wikitext[end:]
	nextRe := regexp.MustCompile(fmt.Sprintf(`(?m)^={2,%d}[^=]`, level))
	if loc := nextRe.FindStringIndex(rest); loc != nil {
		return rest[:loc[0]], level, true
	}
	return rest, level, true
}

// findFormHeadings finds every "=====<form name>=====" sub-heading one
// level deeper than the breeding section's own heading -- Meowth's page has
// three ("Meowth", "Alolan Meowth", "Galarian Meowth") each preceding that
// form's own breeding entries; a species with one form has none at all, and
// its entries just start directly. Reuses templateOccurrence purely as a
// (text, position) pair so it sorts and merges with real template
// occurrences below.
func findFormHeadings(section string, level int) []templateOccurrence {
	eq := regexp.QuoteMeta(strings.Repeat("=", level+1))
	re := regexp.MustCompile(`(?m)^` + eq + `\s*(.+?)\s*` + eq + `\s*$`)
	var out []templateOccurrence
	for _, m := range re.FindAllStringSubmatchIndex(section, -1) {
		out = append(out, templateOccurrence{Name: "form", Body: section[m[2]:m[3]], Start: m[0]})
	}
	return out
}

// buildMoveNameIndex maps a normalized move name to its DB id, built from
// the same PokeAPI English names (and the same cache) the description
// scraper's title resolution already uses -- so for moves already fetched
// during that pass, this costs zero new requests.
//
// Keys are normalized to lowercase-alphanumeric-only, not just lowercased:
// Bulbapedia's older-generation templates are inconsistent about spacing in
// the move-name field ("Grass Whistle" on one page, "GrassWhistle" on
// another, both the same move) -- verified live on Bulbasaur's Generation
// IV vs Generation VI subpages.
// moveNameAliases bridges Bulbapedia's vintage move names -- what a move
// was actually called in-game at the time, still used verbatim on
// older-generation learnset subpages -- to PokeAPI's current name, which is
// all buildMoveNameIndex has to go on (PokeAPI's names[] carries only the
// modern name; verified live, e.g. move 185's only English name is "Feint
// Attack", with no historical variant exposed anywhere in the response).
// Found by running the scraper with -old-gens: every one of these failed
// consistently on Generation II-V subpages and never on VI+, matching the
// real Gen VI (X/Y) move-name overhaul -- Faint Attack, SmellingSalt (one
// word, no space, exactly as it was coded pre-rename), and Hi Jump Kick were
// all renamed to their current spelling starting there.
var moveNameAliases = map[string]string{
	"faint attack": "feint attack",
	"smellingsalt": "smelling salts",
	"hi jump kick": "high jump kick",
}

func buildMoveNameIndex(ctx context.Context, pool *pgxpool.Pool, cache *diskCache) (map[string]int32, []unresolvedItem, error) {
	moves, err := fetchEntities(ctx, pool, movesQuery)
	if err != nil {
		return nil, nil, err
	}
	idx := map[string]int32{}
	var unresolved []unresolvedItem
	for _, m := range moves {
		name, err := fetchPokeAPIName(cache, "move", m.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("move %d name: %w", m.ID, err)
		}
		if name == "" {
			unresolved = append(unresolved, unresolvedf("move %d (%s): no English PokeAPI name", m.ID, m.Name))
			continue
		}
		idx[normalizeMoveKey(name)] = m.ID
	}
	for alias, current := range moveNameAliases {
		if id, ok := idx[normalizeMoveKey(current)]; ok {
			idx[normalizeMoveKey(alias)] = id
		}
	}
	return idx, unresolved, nil
}

var moveKeyRe = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeMoveKey(s string) string {
	return moveKeyRe.ReplaceAllString(strings.ToLower(s), "")
}

// scrapeEggMoves is the top-level entry point. Gen IX comes from the same
// main species pages the description scraper already cached; older
// generations (gated by includeOldGens, since they require ~4,400
// additional, previously-unfetched subpage requests) come from
// "<Species> (Pokémon)/Generation <N> learnset" subpages.
func scrapeEggMoves(ctx context.Context, pool *pgxpool.Pool, limit, batchSize int, cacheDir string, includeOldGens bool) ([]moveDetailRow, []unresolvedItem, error) {
	pageCache, err := newDiskCache(cacheDir + "/bulbapedia")
	if err != nil {
		return nil, nil, err
	}
	pokeCache, err := newDiskCache(cacheDir + "/pokeapi")
	if err != nil {
		return nil, nil, err
	}

	pokemonRows, err := fetchPokemonRows(ctx, pool)
	if err != nil {
		return nil, nil, err
	}
	idx := buildSpeciesIndex(pokemonRows)

	moveIndex, unresolved, err := buildMoveNameIndex(ctx, pool, pokeCache)
	if err != nil {
		return nil, nil, err
	}

	var speciesIDs []int32
	for sid := range idx.forms {
		speciesIDs = append(speciesIDs, sid)
	}
	sort.Slice(speciesIDs, func(i, j int) bool { return speciesIDs[i] < speciesIDs[j] })
	if limit > 0 && len(speciesIDs) > limit {
		speciesIDs = speciesIDs[:limit]
	}

	type pageRef struct {
		title     string
		speciesID int32
		gen       string // "IX" for the main page, "II".."VIII" for a subpage
	}
	var pages []pageRef

	for _, sid := range speciesIDs {
		name, err := fetchPokeAPIName(pokeCache, "pokemon-species", sid)
		if err != nil {
			return nil, nil, fmt.Errorf("species %d name: %w", sid, err)
		}
		if name == "" {
			unresolved = append(unresolved, unresolvedf("species %d: no English PokeAPI name", sid))
			continue
		}
		base := buildTitle(name, "Pokémon")
		pages = append(pages, pageRef{title: base, speciesID: sid, gen: "IX"})
		if includeOldGens {
			for _, g := range []string{"II", "III", "IV", "V", "VI", "VII", "VIII"} {
				pages = append(pages, pageRef{title: base + "/Generation " + g + " learnset", speciesID: sid, gen: g})
			}
		}
	}

	titles := make([]string, len(pages))
	for i, p := range pages {
		titles[i] = p.title
	}

	fetched, err := fetchBulbapediaBatch(pageCache, titles, batchSize)
	if err != nil {
		return nil, nil, err
	}

	var rows []moveDetailRow
	for _, p := range pages {
		page := fetched[p.title]
		if page.Missing {
			// A missing older-gen subpage is expected -- most species didn't
			// exist yet, or never had one split out. Only the main page (IX)
			// missing is worth surfacing.
			if p.gen == "IX" {
				unresolved = append(unresolved, unresolvedf("page not found: %s (species %d)", p.title, p.speciesID))
			}
			continue
		}
		sec, level, ok := breedingSection(page.Content)
		if !ok {
			continue // no breeding section on this page/gen -- not every species breeds
		}
		cfg := genConfigs[p.gen]
		r, u := parseBreedingSection(sec, level, cfg, p.speciesID, idx, p.title, moveIndex)
		rows = append(rows, r...)
		unresolved = append(unresolved, u...)
	}

	return rows, unresolved, nil
}

// parseBreedingSection walks one generation's breeding section in document
// order, merging three kinds of occurrence -- form sub-headings, standalone
// {{gameabbrevN|CODE}} block markers, and {{learnlist/breedN|...}} entries
// -- sorted by position, exactly like parsePokemonDexEntries's Dex/Form
// walk in bulba.go and for the same reason: which form or which game block
// an entry belongs to is determined by what came before it in the document,
// not by anything inside the entry template itself.
func parseBreedingSection(section string, level int, cfg genBreedingConfig, speciesID int32, idx speciesIndex, pageTitle string, moveIndex map[string]int32) ([]moveDetailRow, []unresolvedItem) {
	entryTemplate := "learnlist/breed" + cfg.TemplateSuffix
	markerTemplate := "gameabbrev" + cfg.TemplateSuffix

	forms := findFormHeadings(section, level)
	markers := findTemplates(section, markerTemplate)
	entries := findTemplates(section, entryTemplate)

	type occurrence struct {
		kind string // "form", "marker", "entry"
		text string // form heading text, or marker CODE
		body string // entry template body
		pos  int
	}
	var all []occurrence
	for _, f := range forms {
		all = append(all, occurrence{kind: "form", text: f.Body, pos: f.Start})
	}
	for _, m := range markers {
		all = append(all, occurrence{kind: "marker", text: m.Body, pos: m.Start})
	}
	for _, e := range entries {
		all = append(all, occurrence{kind: "entry", body: e.Body, pos: e.Start})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].pos < all[j].pos })

	var currentIDs []int32
	if base, ok := idx.literalBase[speciesID]; ok {
		currentIDs = []int32{base.ID}
	}
	currentGroups := cfg.DefaultGroups

	var rows []moveDetailRow
	var unresolved []unresolvedItem

	for _, o := range all {
		switch o.kind {
		case "form":
			marker := strings.TrimSpace(o.text)
			if ids, ok := resolveFormMarker(marker, speciesID, idx); ok {
				currentIDs = ids
			} else if reason, known := knownGaps[knownGapKey{speciesID, marker}]; known {
				unresolved = append(unresolved, unresolvedItem{
					Message: fmt.Sprintf("breeding form marker %q on %s (species %d)", marker, pageTitle, speciesID),
					Known:   true, Reason: reason,
				})
				currentIDs = nil
			} else {
				unresolved = append(unresolved, unresolvedf(
					"breeding form marker %q on %s (species %d): no unambiguous match", marker, pageTitle, speciesID))
				currentIDs = nil
			}

		case "marker":
			code := strings.TrimSpace(o.text)
			if grp, ok := gameAbbrevCodeToGroup[code]; ok {
				currentGroups = []string{grp}
			} else {
				unresolved = append(unresolved, unresolvedf(
					"unrecognized breeding game marker %q on %s (species %d)", code, pageTitle, speciesID))
				currentGroups = nil // stop emitting until a recognized marker appears
			}

		case "entry":
			if len(currentIDs) == 0 {
				continue
			}
			parts := splitTopLevel(o.body)
			if len(parts) < 2 {
				continue
			}
			moveName := strings.TrimSpace(parts[1])
			moveID, ok := moveIndex[normalizeMoveKey(moveName)]
			if !ok {
				unresolved = append(unresolved, unresolvedf(
					"unknown move %q in breeding entry on %s (species %d)", moveName, pageTitle, speciesID))
				continue
			}

			groups := currentGroups
			if cfg.EntryOverride != nil {
				for _, p := range parts {
					if grp, ok := cfg.EntryOverride[strings.TrimSpace(p)]; ok {
						groups = []string{grp}
						break
					}
				}
			}
			for _, id := range currentIDs {
				for _, g := range groups {
					rows = append(rows, moveDetailRow{
						PokemonID: id, MoveID: moveID, Version: g,
						Source: "bulbapedia", SourceTitle: pageTitle,
					})
				}
			}
		}
	}
	return rows, unresolved
}
