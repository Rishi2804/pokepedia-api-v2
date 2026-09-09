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

	// FormLabel is set when the entry carries a trailing {{tt|*|<Label>}}
	// tag disambiguating which specific pokemon row this ONE row applies
	// to -- distinct from a form heading (see findFormHeadings), which
	// applies to every entry until the next one. Verified live: Dialga and
	// Palkia's Origin Forme signature moves (Roar of Time, Spacial Rend)
	// and Giratina's Altered/Origin Shadow Force are tagged this way with
	// no heading anywhere on the page, and Rotom's five appliance-exclusive
	// moves are tagged the same way alongside an otherwise fully shared,
	// untagged moveset. Resolved per-entry via resolveFormMarker, not
	// folded into currentIDs state, since an untagged entry immediately
	// after a tagged one must NOT inherit the tag.
	FormLabel *string
}

// formTagRe extracts a trailing {{tt|*|<Label>}} tag's label. The same
// {{tt|*|...}} syntax also marks a availability note ("Version 2.0.0
// onwards") -- but always embedded INSIDE the Learn field (parseLearnLevel
// already strips it there), never as a separate trailing field; verified by
// scanning every levelLA/tutorPLA/levelZA/tmZA occurrence in the cache, so
// any trailing occurrence this matches is safe to treat as a form label.
var formTagRe = regexp.MustCompile(`\{\{tt\|\*\|([^}]*)\}\}`)

func extractFormTag(s string) *string {
	m := formTagRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	label := strings.TrimSpace(m[1])
	return &label
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
	var formLabel *string
	if len(parts) > 11 {
		formLabel = extractFormTag(parts[11])
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
		FormLabel:   formLabel,
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
	var formLabel *string
	if len(parts) > 9 {
		formLabel = extractFormTag(parts[9])
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
		FormLabel:   formLabel,
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
	var formLabel *string
	if len(parts) > 7 {
		formLabel = extractFormTag(parts[7])
	}
	return legendsEntryFields{
		MoveName:    strings.TrimSpace(parts[2]),
		Method:      "level-up",
		Level:       learn,
		SecondLevel: plus,
		PowerBase:   parseNullableInt(parts[5]),
		Cooldown:    cd,
		FormLabel:   formLabel,
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
	var formLabel *string
	if len(parts) > 6 {
		formLabel = extractFormTag(parts[6])
	}
	return legendsEntryFields{
		MoveName:  strings.TrimSpace(parts[1]),
		Method:    "machine",
		Level:     0,
		PowerBase: parseNullableInt(parts[4]),
		Cooldown:  cd,
		FormLabel: formLabel,
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

// legendsSpeciesDefaults gives the initial currentIDs for a species' Legends
// section when idx.literalBase either doesn't exist or is wrong, verified
// case by case rather than inferred from a general rule -- no general rule
// survived contact with the data. A "first/lowest id" or "PokeAPI is_default
// variety" heuristic both get Giratina wrong: PokeAPI marks giratina-altered
// (id 487) as the default variety species-wide, yet Legends: Arceus's own
// moveset belongs entirely to giratina-origin (10007) -- confirmed two ways:
// PokeAPI's own per-pokemon move data lists 10 Legends: Arceus moves for
// giratina-origin and zero for giratina-altered, and it matches Legends:
// Arceus's actual plot (Giratina is caught at Turnback Cave in Origin
// Forme). Every other entry here was individually verified against either
// PokeAPI's move data (where it exists -- Legends: Arceus only) or this
// database's own existing mainline-game movedetails counts (where it
// doesn't -- Legends: Z-A):
//
//   - Tornadus/Thundurus/Landorus/Basculegion/Enamorus: PokeAPI shows 9
//     Legends: Arceus moves on the Incarnate/male variety, 0 on the other.
//   - Aegislash/Meloetta/Morpeko: Stance Change/Relic Song/Hunger Switch are
//     battle-only transformation states with no independent existence
//     outside battle, and this database's existing mainline data already
//     stores an identical moveset for both rows of each (Aegislash
//     shield/blade: 50 sword-shield rows each; Meloetta aria/pirouette: 546
//     rows each across every version; Morpeko full-belly/hangry: 67
//     sword-shield rows each) -- so both get every untagged Legends entry.
//   - Zygarde: its three non-Mega formes already carry an identical
//     49-row sun-moon moveset each; zygarde-mega is added separately by
//     zaMegaByBase below, following this database's own established
//     convention that a Mega Evolution always gets a full copy of its base
//     form's moveset (verified: charizard-mega-x alone carries 342 mainline
//     movedetails rows).
//   - Rotom: idx.literalBase resolves to plain "rotom" alone, which is
//     wrong here -- this database's existing mainline data shows all six
//     Rotom forms (base + five appliances) sharing an identical 50-row
//     sword-shield moveset, so the untagged majority of Rotom's Legends
//     entries need all six; the five appliance-exclusive signature moves
//     each carry their own {{tt|*|<Appliance> Rotom}} tag (see FormLabel)
//     that overrides this default for just that one entry.
//
// Keldeo and Shaymin are deliberately absent: existing mainline data shows
// keldeo-ordinary/keldeo-resolute (459 vs 409 rows) and shaymin-land/
// shaymin-sky (438 vs 397 rows, on the Legends: Arceus side -- Shaymin's
// level-up section DOES have "Land Forme"/"Sky Forme" headings, but its
// tutor section has none) have different-sized movesets, so assuming
// parity would risk writing wrong data, and there is no independent source
// (PokeAPI has no Legends: Z-A data at all, and its Legends: Arceus tutor
// coverage is Hisuian-forms-only per this tool's own doc comment) to verify
// which -- or both -- the untagged entries actually mean. Both left as
// reported, known gaps (see knowngaps.go) rather than a guess.
var legendsSpeciesDefaults = map[int32][]int32{
	487: {10007},                                  // Giratina -> origin only
	641: {641},                                    // Tornadus -> incarnate only
	642: {642},                                    // Thundurus -> incarnate only
	645: {645},                                    // Landorus -> incarnate only
	902: {902},                                    // Basculegion -> male only
	905: {905},                                    // Enamorus -> incarnate only
	681: {681, 10026},                             // Aegislash -> shield, blade
	648: {648, 10018},                             // Meloetta -> aria, pirouette
	877: {877, 10187},                             // Morpeko -> full-belly, hangry
	718: {718, 10120, 10181},                      // Zygarde -> 50%, Complete, 10% (mega via zaMegaByBase)
	479: {479, 10008, 10009, 10010, 10011, 10012}, // Rotom -> base + 5 appliances

	// Toxtricity's Legends: Z-A TM section has no form headings at all
	// (unlike its level-up section, which starts immediately with an
	// "Amped Form" heading before any entry -- so this default is only
	// ever actually used by the TM section). Verified via existing
	// mainline data: toxtricity-amped and toxtricity-low-key already carry
	// an identical 147-row moveset each, so both get every untagged TM
	// entry too.
	849: {849, 10184}, // Toxtricity -> amped, low-key
}

// zaMegaByBase maps a species' "normal" pokemon id(s) to the Z-A-exclusive
// Mega Evolution row(s) 000007_legends_za_data.up.sql added for it (ids
// 10278-10326, 49 rows total) -- applied only for legends-za, since
// Legends: Arceus has no Mega Evolution mechanic at all and every OTHER
// pre-existing "-mega"/"-mega-x"/"-mega-y" row in this database (ids below
// 10278, e.g. charizard-mega-x) is confirmed NOT legal in Legends: Z-A: none
// of their species appear in pokepedia-battle/src/formats.ts's
// ZA_MEGA_STONE_IDS, the authoritative list this mapping was cross-checked
// against. Mega Evolution never changes movepool in any generation, and
// Bulbapedia's own Z-A learnset pages confirm this -- e.g. Charizard's Z-A
// section lists one unified moveset with no Mega-specific carve-out -- so
// every untagged (or heading/tag-resolved) entry for a mapped base id also
// belongs to its Mega row(s). Gigantamax rows are never included: Legends:
// Z-A has no Dynamax/Gigantamax mechanic.
var zaMegaByBase = map[int32][]int32{
	36:    {10278},        // clefable
	71:    {10279},        // victreebel
	121:   {10280},        // starmie
	149:   {10281},        // dragonite
	154:   {10282},        // meganium
	160:   {10283},        // feraligatr
	227:   {10284},        // skarmory
	478:   {10285},        // froslass
	500:   {10286},        // emboar
	530:   {10287},        // excadrill
	545:   {10288},        // scolipede
	560:   {10289},        // scrafty
	604:   {10290},        // eelektross
	609:   {10291},        // chandelure
	652:   {10292},        // chesnaught
	655:   {10293},        // delphox
	658:   {10294},        // greninja (NOT greninja-ash -- see below)
	668:   {10295},        // pyroar
	670:   {10296},        // floette
	687:   {10297},        // malamar
	689:   {10298},        // barbaracle
	691:   {10299},        // dragalge
	701:   {10300},        // hawlucha
	718:   {10301},        // zygarde-50
	780:   {10302},        // drampa
	870:   {10303},        // falinks
	26:    {10304, 10305}, // raichu -> mega-x, mega-y
	358:   {10306},        // chimecho
	359:   {10307},        // absol -> mega-z only (classic absol-mega is not Z-A legal)
	398:   {10308},        // staraptor
	445:   {10309},        // garchomp -> mega-z only
	448:   {10310},        // lucario -> mega-z only
	485:   {10311},        // heatran
	491:   {10312},        // darkrai
	623:   {10313},        // golurk
	678:   {10314},        // meowstic-male
	740:   {10315},        // crabominable
	768:   {10316},        // golisopod
	801:   {10317},        // magearna
	10147: {10318},        // magearna-original
	807:   {10319},        // zeraora
	952:   {10320},        // scovillain
	970:   {10321},        // glimmora
	978:   {10322},        // tatsugiri (curly, the default forme)
	10258: {10323},        // tatsugiri-droopy
	10259: {10324},        // tatsugiri-stretchy
	998:   {10325},        // baxcalibur
	10025: {10326},        // meowstic-female
	// greninja-ash (10117) is deliberately absent, same reasoning as
	// Keldeo: existing mainline data shows it does NOT mirror plain
	// greninja (199 vs 372 rows), so parity can't be assumed, and nothing
	// independently confirms whether Ash-Greninja carries Z-A's Mega
	// Evolution mechanic at all (Battle Bond and Mega Evolution may not be
	// combinable in-game). Left unmirrored rather than guessed.
}

// expandWithZAMegas adds each id's Z-A-exclusive Mega row(s), if any, to
// the target set for one legends-za entry -- see zaMegaByBase. A no-op for
// every id without a mapped Mega and for every legends-arceus row (callers
// only invoke this when version == "legends-za").
func expandWithZAMegas(ids []int32) []int32 {
	out := append([]int32{}, ids...)
	for _, id := range ids {
		out = append(out, zaMegaByBase[id]...)
	}
	return out
}

// parseLegendsSection walks one method section in document order, merging
// level-5 form sub-headings with occurrences of entryTemplate -- the exact
// same positional-state pattern parseBreedingSection uses, and for the same
// reason: which form an entry belongs to is determined by what heading came
// before it, not by anything inside the entry template itself.
func parseLegendsSection(section string, speciesID int32, idx speciesIndex, pageTitle string, moveIndex map[string]int32, entryTemplate, version string, parseFields func([]string) (legendsEntryFields, bool)) ([]legendsMoveRow, []unresolvedItem) {
	forms := findFormHeadings(section, 4)
	entries := findTemplates(section, entryTemplate)
	// {{gameabbrevN|...}} markers are tracked purely to decide when to
	// reset currentIDs -- see defaultIDs below. The code value itself (LA,
	// SwSh, BDSP, SV, ZA, ...) never needs checking: ANY marker means a new
	// game's block is starting, and this method section only ever contains
	// entryTemplate occurrences for ONE game anyway.
	markers := findTemplates(section, "gameabbrev8", "gameabbrev9")

	type occurrence struct {
		kind string // "form", "marker", "entry"
		text string
		body string
		pos  int
	}
	var all []occurrence
	for _, f := range forms {
		all = append(all, occurrence{kind: "form", text: f.Body, pos: f.Start})
	}
	for _, m := range markers {
		all = append(all, occurrence{kind: "marker", pos: m.Start})
	}
	for _, e := range entries {
		all = append(all, occurrence{kind: "entry", body: e.Body, pos: e.Start})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].pos < all[j].pos })

	// defaultIDs is what currentIDs resets to at a game-block boundary --
	// but ONLY when 2+ form headings occurred since the PREVIOUS marker
	// (see headingsSinceMarker below), never for 0 or 1. Verified against
	// three different real page shapes:
	//
	//   - Wormadam/Meowstic: one heading ("Plant Cloak", "Male Meowstic")
	//     is immediately followed by TWO markers in sequence (its own
	//     mainline block, then its own Legends block), each with the
	//     heading's own footer closing the FIRST one -- exactly 1 heading
	//     since the previous marker (section start, in this case) when the
	//     Legends marker is reached, so no reset: the heading correctly
	//     spans both of "its" blocks.
	//   - Basculin: "White-Striped Basculin" is the only heading between
	//     the SwSh marker and the LA marker -- exactly 1, so no reset: the
	//     heading is meant for the upcoming LA block specifically.
	//   - Shaymin: "Land Forme" AND "Sky Forme" both occur between the
	//     BDSP marker and the LA marker -- 2 headings splitting BDSP's OWN
	//     data by form, neither of which says anything about the LA block
	//     that follows with no heading of its own. Without resetting here,
	//     every LA move was silently attributed to shaymin-sky alone
	//     (PokeAPI confirms both formes have real, DIFFERENT Legends:
	//     Arceus movesets -- 22 vs 14 moves -- so that's not incomplete,
	//     it's wrong); a naive "any footer before the marker" version of
	//     this check got this case right but broke Wormadam/Meowstic
	//     (their mainline sub-block's footer closes before the Legends
	//     marker too, yet the heading must still carry through) -- the
	//     count of headings, not the presence of a footer, is what
	//     actually distinguishes "this marker still belongs to the last
	//     heading" from "the last marker's own data was internally
	//     form-split, and none of those headings represents this one".
	defaultIDs, hasDefault := legendsSpeciesDefaults[speciesID]
	if !hasDefault {
		if base, ok := idx.literalBase[speciesID]; ok {
			defaultIDs = []int32{base.ID}
		}
	}
	currentIDs := defaultIDs
	headingsSinceMarker := 0

	var rows []legendsMoveRow
	var unresolved []unresolvedItem
	sawEmptySkip := false

	for _, o := range all {
		switch o.kind {
		case "marker":
			if headingsSinceMarker >= 2 {
				currentIDs = defaultIDs
			}
			headingsSinceMarker = 0
			continue
		case "form":
			headingsSinceMarker++
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

		// A per-entry {{tt|*|<Label>}} tag (see FormLabel) overrides
		// currentIDs for JUST this row -- it never becomes the new
		// currentIDs, since the very next, untagged entry must not inherit
		// it (verified live: Dialga's Roar of Time is the only tagged
		// entry on its whole page; every other move is untagged and
		// correctly applies to plain "dialga" alone).
		var targetIDs []int32
		if fields.FormLabel != nil {
			if ids, ok := resolveFormMarker(*fields.FormLabel, speciesID, idx); ok {
				targetIDs = ids
			} else if reason, known := knownGaps[knownGapKey{speciesID, *fields.FormLabel}]; known {
				unresolved = append(unresolved, unresolvedItem{
					Message: fmt.Sprintf("%s form tag %q on %s (species %d)", entryTemplate, *fields.FormLabel, pageTitle, speciesID),
					Known:   true, Reason: reason,
				})
				continue
			} else {
				unresolved = append(unresolved, unresolvedf(
					"%s form tag %q on %s (species %d): no unambiguous match", entryTemplate, *fields.FormLabel, pageTitle, speciesID))
				continue
			}
		} else {
			if len(currentIDs) == 0 {
				sawEmptySkip = true
				continue
			}
			targetIDs = currentIDs
		}

		if version == "legends-za" {
			targetIDs = expandWithZAMegas(targetIDs)
		}

		for _, pid := range targetIDs {
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

	// A page whose untagged entries never got a currentIDs to belong to at
	// all (no literalBase, no legendsSpeciesDefaults entry, no heading ever
	// appeared) is reported once here rather than once per skipped entry --
	// matching printUnresolved's own known/unknown split, so a diagnosed
	// case like Keldeo (see legendsSpeciesDefaults) collapses to a one-line
	// count instead of repeating for every one of its ~15-20 entries.
	if sawEmptySkip {
		const noFormMarker = "(no form heading, tag, or default)"
		if reason, known := knownGaps[knownGapKey{speciesID, noFormMarker}]; known {
			unresolved = append(unresolved, unresolvedItem{
				Message: fmt.Sprintf("%s entries on %s (species %d): no way to attribute untagged rows", entryTemplate, pageTitle, speciesID),
				Known:   true, Reason: reason,
			})
		} else {
			unresolved = append(unresolved, unresolvedf(
				"%s entries on %s (species %d): no form heading, tag, or default -- untagged rows skipped", entryTemplate, pageTitle, speciesID))
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
