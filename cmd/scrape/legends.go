package main

// legends.go is cmd/scrape's Legends: Arceus / Legends: Z-A learnset pass.
// Both games have zero movedetails rows today, and their move data is
// shaped differently from mainline -- see legendsMoveRow in types.go and
// 000010_legends_move_values.up.sql for why a new table exists at all.
//
// PokeAPI is not usable as the source for either: it has Legends: Arceus
// level-up moves, but its tutor moves exist only for Hisuian forms (regular
// species are missing them entirely -- verified live, Pikachu 0 vs
// Bulbapedia's 12), and it has nothing at all for Legends: Z-A.
//
// Bulbapedia embeds both games' learnsets INLINE within the same
// "By leveling up" / "By TM" / "By tutoring" sections mainline entries
// already live in, gated by a {{gameabbrevN|LA}} / {{gameabbrevN|ZA}} block
// marker and a dedicated entry template -- verified live:
//
//   - Legends: Arceus lives on the same "<Species> (Pokemon)/Generation
//     VIII learnset" subpages the egg-move pass already cached with
//     -old-gens, inside "By leveling up" ({{learnlist/levelLA|...}}) and
//     "By tutoring" ({{learnlist/tutorPLA|...}}).
//   - Legends: Z-A lives on the main "<Species> (Pokemon)" page, inside
//     "By leveling up" ({{learnlist/levelZA|...}}) and "By TM"
//     ({{learnlist/tmZA|...}}).
//
// Multi-form species repeat the exact same level-5 form sub-headings
// (e.g. =====Alolan Meowth=====) inside each method section that the
// egg-move breeding parser already handles, so this reuses the same
// document-order occurrence-merge pattern as parseBreedingSection.
//
// The entry template name itself (levelLA vs level8, tmZA vs tm9, ...)
// unambiguously identifies which game a row belongs to -- unlike the
// breeding pass, this parser never needs to track the {{gameabbrevN|...}}
// marker as state; it exists on the page purely as a Bulbapedia display
// divider.

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// legendsEntryFields is what one template's positional args parse into,
// before PokemonID/Version/Source are filled in by the generic walker.
type legendsEntryFields struct {
	MoveName    string
	Method      string
	Level       int32
	SecondLevel *int32
	PowerBase   *int32
	PowerStrong *int32
	PowerAgile  *int32
	Accuracy1   *int32
	Accuracy2   *int32
	PP          *int32
	Cooldown    *int32
}

// parseNullableInt parses one Bulbapedia stat field. Both the raw wikitext
// entity (&mdash;) and the literal rendered em dash (—) mean "no value" --
// status moves have no power, and neither Legends game shows a second
// accuracy value for moves that have none at all (verified: Acid Armor's
// LA row is power/accuracy all dashes). An unparseable non-dash value
// returns nil too; the caller decides whether that specific field is
// allowed to be nil or signals a malformed row.
func parseNullableInt(s string) *int32 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&mdash;", "—")
	if s == "" || s == "—" || s == "-" {
		return nil
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		return nil
	}
	v := int32(n)
	return &v
}

func parseRequiredInt(s string) (int32, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return int32(n), true
}

// parseLearnLevel parses a levelLA/levelZA Learn field. Most rows are a
// plain number, but Bulbapedia tags two special cases with a nested
// {{tt|...}} tooltip instead of a level: "Evo." (learned automatically on
// evolving into this Pokemon -- verified against this database's own
// existing Scarlet/Violet import, which already stores evolution-triggered
// level-up moves as level_learned=0, e.g. Charizard's Air Slash) and "Rem."
// (learnable only via the Move Reminder, for a move the pre-evolution
// learns at level 1 that this already-evolved Pokemon can no longer level
// into normally). Both map to the same level_learned=0 sentinel this
// database already uses for every method with no specific level (tutor,
// machine, egg) -- there is no dedicated "evolution" or "move reminder"
// method in public.learnmethod, and inventing one for two Bulbapedia-only
// tags neither PokeAPI nor this schema otherwise distinguishes would be
// unjustified scope. A trailing, unrelated tooltip -- {{tt|*|Version 2.0.0
// onwards}}, marking a post-launch addition -- can follow a real numeric
// level ("1{{tt|*|...}}"); stripping everything from the first "{{" onward
// before parsing handles that case as well as the tag-only ones.
func parseLearnLevel(s string) int32 {
	if i := strings.Index(s, "{{"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return int32(n)
}

// parseSecondLevel parses a levelLA Mastery field (or levelZA Plus field,
// which has never been observed doing this but is parsed the same way
// defensively). Verified live: Gyarados's LA Splash entry is
// "{{learnlist/levelLA|1|None|Splash|...}}" -- Splash has no Strong/Agile
// Style benefit at all, so Bulbapedia writes the literal word "None" rather
// than a level.
func parseSecondLevel(s string) *int32 {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "none") {
		return nil
	}
	return parseNullableInt(s)
}

// levelLA field layout, verified against Bulbapedia's own
// Template:Learnlist/levelh/LA column headers:
//
//	Learn | Mastery | Move | Type | Cat | Power(Base) | Power(Strong) |
//	Power(Agile) | Accuracy | Accuracy(Strong/Agile) | PP
//
// PP is required for every real entry (every move has one, including
// Status moves), so a missing/blank PP field is the signature of a
// silently-shifted row -- see parseTutorPLAFields for the case this
// actually catches (Phione's Zen Headbutt tutor entry).
func parseLevelLAFields(parts []string) (legendsEntryFields, bool) {
	if len(parts) < 11 {
		return legendsEntryFields{}, false
	}
	learn := parseLearnLevel(parts[0])
	mastery := parseSecondLevel(parts[1])
	pp := parseNullableInt(parts[10])
	if pp == nil {
		return legendsEntryFields{}, false
	}
	return legendsEntryFields{
		MoveName:    strings.TrimSpace(parts[2]),
		Method:      "level-up",
		Level:       learn,
		SecondLevel: mastery,
		PowerBase:   parseNullableInt(parts[5]),
		PowerStrong: parseNullableInt(parts[6]),
		PowerAgile:  parseNullableInt(parts[7]),
		Accuracy1:   parseNullableInt(parts[8]),
		Accuracy2:   parseNullableInt(parts[9]),
		PP:          pp,
	}, true
}

// tutorPLA field layout: same as levelLA minus the leading Learn/Mastery
// pair (tutor moves have no level). This is the template Phione's
// Zen Headbutt entry is malformed on -- its fields are shifted one position
// left (missing PowerStrong), which this function catches generically via
// the same "PP must be present" check levelLA uses, rather than a
// Phione-specific special case: the shift leaves PP's slot (parts[8])
// blank, exactly what an ordinary missing-field bug looks like anywhere
// else this template is used correctly.
func parseTutorPLAFields(parts []string) (legendsEntryFields, bool) {
	if len(parts) < 9 {
		return legendsEntryFields{}, false
	}
	pp := parseNullableInt(parts[8])
	if pp == nil {
		return legendsEntryFields{}, false
	}
	return legendsEntryFields{
		MoveName:    strings.TrimSpace(parts[0]),
		Method:      "tutor",
		Level:       0,
		PowerBase:   parseNullableInt(parts[3]),
		PowerStrong: parseNullableInt(parts[4]),
		PowerAgile:  parseNullableInt(parts[5]),
		Accuracy1:   parseNullableInt(parts[6]),
		Accuracy2:   parseNullableInt(parts[7]),
		PP:          pp,
	}, true
}

// levelZA field layout, verified against Template:Learnlist/levelh/ZA:
// Learn | Plus | Move | Type | Cat | Power | CD. CD (cooldown) replaces PP
// entirely in this game, so it plays the same "must be present" role PP
// plays for levelLA/tutorPLA above.
func parseLevelZAFields(parts []string) (legendsEntryFields, bool) {
	if len(parts) < 7 {
		return legendsEntryFields{}, false
	}
	learn := parseLearnLevel(parts[0])
	plus := parseSecondLevel(parts[1])
	cd := parseNullableInt(parts[6])
	if cd == nil {
		return legendsEntryFields{}, false
	}
	return legendsEntryFields{
		MoveName:    strings.TrimSpace(parts[2]),
		Method:      "level-up",
		Level:       learn,
		SecondLevel: plus,
		PowerBase:   parseNullableInt(parts[5]),
		Cooldown:    cd,
	}, true
}

// tmZA field layout: TM# | Move | Type | Cat | Power | CD -- the move name
// is field 2 here, unlike levelZA where it's field 3, since a TM row has no
// Learn/Plus pair.
func parseTmZAFields(parts []string) (legendsEntryFields, bool) {
	if len(parts) < 6 {
		return legendsEntryFields{}, false
	}
	cd := parseNullableInt(parts[5])
	if cd == nil {
		return legendsEntryFields{}, false
	}
	return legendsEntryFields{
		MoveName:  strings.TrimSpace(parts[1]),
		Method:    "machine",
		Level:     0,
		PowerBase: parseNullableInt(parts[4]),
		Cooldown:  cd,
	}, true
}

// levelUpHeadingRe/tutoringHeadingRe/tmHeadingRe match the level-4 method
// headings both the Generation VIII learnset subpages (Legends: Arceus) and
// the main species pages (Legends: Z-A) use -- verified identical text in
// both places. "By [[TM]]" is suffixed with "/[[TR]]" on pages that predate
// TRs being dropped (Sword/Shield-era subpages); Legends: Z-A's own main-page
// section never carries that suffix, so it's optional.
var levelUpHeadingRe = regexp.MustCompile(`(?m)^====\s*By \[\[Level\|leveling up\]\]\s*====\s*$`)
var tutoringHeadingRe = regexp.MustCompile(`(?m)^====\s*By \[\[Move Tutor\|tutoring\]\]\s*====\s*$`)
var tmHeadingRe = regexp.MustCompile(`(?m)^====\s*By \[\[TM\]\](?:/\[\[TR\]\])?\s*====\s*$`)

// methodSection extracts one level-4 method section's body, bounded by the
// next heading at level 4 or shallower -- mirroring breedingSection's own
// boundary logic in eggmoves.go, just at a fixed level instead of a
// discovered one, since these headings are always "====" on every page this
// tool has seen.
func methodSection(content string, headingRe *regexp.Regexp) (string, bool) {
	loc := headingRe.FindStringIndex(content)
	if loc == nil {
		return "", false
	}
	rest := content[loc[1]:]
	nextRe := regexp.MustCompile(`(?m)^={2,4}[^=]`)
	if m := nextRe.FindStringIndex(rest); m != nil {
		return rest[:m[0]], true
	}
	return rest, true
}

// parseLegendsSection walks one method section in document order, merging
// level-5 form sub-headings with occurrences of entryTemplate -- the exact
// same positional-state pattern parseBreedingSection uses, and for the same
// reason: which form an entry belongs to is determined by what heading came
// before it, not by anything inside the entry template itself.
func parseLegendsSection(section string, speciesID int32, idx speciesIndex, pageTitle string, moveIndex map[string]int32, entryTemplate, version string, parseFields func([]string) (legendsEntryFields, bool)) ([]legendsMoveRow, []unresolvedItem) {
	forms := findFormHeadings(section, 4)
	entries := findTemplates(section, entryTemplate)

	type occurrence struct {
		isForm bool
		text   string
		body   string
		pos    int
	}
	var all []occurrence
	for _, f := range forms {
		all = append(all, occurrence{isForm: true, text: f.Body, pos: f.Start})
	}
	for _, e := range entries {
		all = append(all, occurrence{body: e.Body, pos: e.Start})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].pos < all[j].pos })

	var currentIDs []int32
	if base, ok := idx.literalBase[speciesID]; ok {
		currentIDs = []int32{base.ID}
	}

	var rows []legendsMoveRow
	var unresolved []unresolvedItem

	for _, o := range all {
		if o.isForm {
			marker := strings.TrimSpace(o.text)
			if ids, ok := resolveFormMarker(marker, speciesID, idx); ok {
				currentIDs = ids
			} else if reason, known := knownGaps[knownGapKey{speciesID, marker}]; known {
				unresolved = append(unresolved, unresolvedItem{
					Message: fmt.Sprintf("%s form marker %q on %s (species %d)", entryTemplate, marker, pageTitle, speciesID),
					Known:   true, Reason: reason,
				})
				currentIDs = nil
			} else {
				unresolved = append(unresolved, unresolvedf(
					"%s form marker %q on %s (species %d): no unambiguous match", entryTemplate, marker, pageTitle, speciesID))
				currentIDs = nil
			}
			continue
		}

		if len(currentIDs) == 0 {
			continue
		}
		parts := splitTopLevel(o.body)
		fields, ok := parseFields(parts)
		if !ok {
			unresolved = append(unresolved, unresolvedf(
				"malformed {{%s}} entry on %s (species %d): %s", entryTemplate, pageTitle, speciesID, o.body))
			continue
		}
		moveID, ok := moveIndex[normalizeMoveKey(fields.MoveName)]
		if !ok {
			unresolved = append(unresolved, unresolvedf(
				"unknown move %q in {{%s}} entry on %s (species %d)", fields.MoveName, entryTemplate, pageTitle, speciesID))
			continue
		}
		for _, pid := range currentIDs {
			rows = append(rows, legendsMoveRow{
				PokemonID: pid, MoveID: moveID, Version: version,
				Method: fields.Method, Level: fields.Level,
				SecondLevel: fields.SecondLevel,
				PowerBase:   fields.PowerBase, PowerStrong: fields.PowerStrong, PowerAgile: fields.PowerAgile,
				Accuracy1: fields.Accuracy1, Accuracy2: fields.Accuracy2, PP: fields.PP, Cooldown: fields.Cooldown,
				Source: "bulbapedia", SourceTitle: pageTitle,
			})
		}
	}
	return rows, unresolved
}

// scrapeLegends is the top-level entry point. Legends: Arceus data comes
// from the same Generation VIII learnset subpages -old-gens already
// fetched for egg moves; Legends: Z-A data comes from the main species
// pages the description pass already cached. Both are titled identically to
// how eggmoves.go and bulba.go already build these titles, so this costs
// zero new network requests against a warm cache.
func scrapeLegends(ctx context.Context, pool *pgxpool.Pool, limit, batchSize int, cacheDir string) ([]legendsMoveRow, []unresolvedItem, error) {
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
		game      string // "LA" or "ZA"
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
		pages = append(pages, pageRef{title: base, speciesID: sid, game: "ZA"})
		pages = append(pages, pageRef{title: base + "/Generation VIII learnset", speciesID: sid, game: "LA"})
	}

	titles := make([]string, len(pages))
	for i, p := range pages {
		titles[i] = p.title
	}

	fetched, err := fetchBulbapediaBatch(pageCache, titles, batchSize)
	if err != nil {
		return nil, nil, err
	}

	var rows []legendsMoveRow
	for _, p := range pages {
		page := fetched[p.title]
		if page.Missing {
			// A missing Generation VIII learnset subpage is expected -- most
			// species were never split out that way. A missing main species
			// page is not: every real species has one (it's how every other
			// pass in this tool already sourced its data), so that's worth
			// surfacing.
			if p.game == "ZA" {
				unresolved = append(unresolved, unresolvedf("page not found: %s (species %d)", p.title, p.speciesID))
			}
			continue
		}

		switch p.game {
		case "LA":
			if sec, ok := methodSection(page.Content, levelUpHeadingRe); ok {
				r, u := parseLegendsSection(sec, p.speciesID, idx, p.title, moveIndex, "learnlist/levelLA", "legends-arceus", parseLevelLAFields)
				rows = append(rows, r...)
				unresolved = append(unresolved, u...)
			}
			if sec, ok := methodSection(page.Content, tutoringHeadingRe); ok {
				r, u := parseLegendsSection(sec, p.speciesID, idx, p.title, moveIndex, "learnlist/tutorPLA", "legends-arceus", parseTutorPLAFields)
				rows = append(rows, r...)
				unresolved = append(unresolved, u...)
			}
		case "ZA":
			if sec, ok := methodSection(page.Content, levelUpHeadingRe); ok {
				r, u := parseLegendsSection(sec, p.speciesID, idx, p.title, moveIndex, "learnlist/levelZA", "legends-za", parseLevelZAFields)
				rows = append(rows, r...)
				unresolved = append(unresolved, u...)
			}
			if sec, ok := methodSection(page.Content, tmHeadingRe); ok {
				r, u := parseLegendsSection(sec, p.speciesID, idx, p.title, moveIndex, "learnlist/tmZA", "legends-za", parseTmZAFields)
				rows = append(rows, r...)
				unresolved = append(unresolved, u...)
			}
		}
	}

	return rows, unresolved, nil
}
