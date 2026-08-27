package main

import (
	"strings"
	"testing"
)

// thunderboltDescriptionFixture is the real, unmodified "==Description=="
// section wikitext from Bulbapedia's Thunderbolt (move) page (fetched
// 2026-08-25). It is the regression fixture for the nested-brace bug: a
// naive non-greedy regex on {{movedescentry|...|...}} truncates the games
// field at the first "}}", which belongs to the nested {{gameabbrev9|ZA}}
// template, and would make every ZA check below silently fail.
const thunderboltDescriptionFixture = `{{movedesc|electric}}
{{movedescentry|{{gameabbrevss|Stad}}{{gameabbrevss|Stad2}}|An <sc>Electric</sc>-type attack. Has a one-in-ten chance of paralyzing the target.}}
{{movedescentry|{{gameabbrev2|GSC}}|An attack that may cause paralysis.}}
{{movedescentry|{{gameabbrev3|RSE}}{{gameabbrevss|Colo}}{{gameabbrevss|XD}}|A strong electrical attack that may paralyze the foe.{{sup/3|RS}}{{sup/3|E}} <small>(move)</small><br>A strong electrical attack. It may paralyze the target.{{sup/ss|Colo}}{{sup/ss|XD}} <small>(move)</small><br>A powerful electric attack that may cause paralysis. <small>(TM)</small>}}
{{movedescentry|{{gameabbrev3|FRLG}}|A strong electrical attack that may also leave the foe paralyzed.}}
{{movedescentry|{{gameabbrev4|DPPtHGSS}}{{gameabbrevss|PBR}}|A strong electric blast is loosed at the foe. It may also leave the foe paralyzed.}}
{{movedescentry|{{gameabbrev5|BWB2W2}}|A strong electric blast is loosed at the target. It may also leave the target with paralysis.}}
{{movedescentry|{{gameabbrev6|XYORAS}}<br>{{gameabbrev7|SMUSUMPE}}<br>{{gameabbrev8|SwShBDSP}}|A strong electric blast crashes down on the target. This may also leave the target with paralysis.}}
{{movedescentry|{{gameabbrev8|LA}}<br>{{gameabbrev9|SV}}|The user attacks the target with a strong electric blast. This may also leave the target with paralysis.}}
{{movedescentry|{{gameabbrev9|ZA}}|The user attacks with a strong electric blast. This may also leave targets paralyzed.}}
{{movedescentry|{{gameabbrevss|CHP}}|Has a 10% chance of paralyzing the target.}}
|}
|}{{left clear}}`

// findZAOccurrence locates the movedescentry whose games field contains a
// {{gameabbrev9|ZA}} token -- by content, not position, since the fixture
// also has a trailing Champions ({{gameabbrevss|CHP}}) entry after it and a
// position-based index would be one fixture edit away from silently testing
// the wrong entry.
func findZAOccurrence(t *testing.T, occs []templateOccurrence) (gamesField, text string) {
	t.Helper()
	for _, occ := range occs {
		parts := splitTopLevel(occ.Body)
		if len(parts) != 2 {
			continue
		}
		for _, code := range gameAbbrevCodes(parts[0]) {
			if code == "ZA" {
				return parts[0], strings.TrimSpace(parts[1])
			}
		}
	}
	t.Fatal("no movedescentry with a ZA game code found")
	return "", ""
}

func TestFindTemplates_MoveDescEntry_NestedBraces(t *testing.T) {
	occs := findTemplates(thunderboltDescriptionFixture, "movedescentry")
	if len(occs) != 10 {
		t.Fatalf("got %d movedescentry occurrences, want 10", len(occs))
	}

	// The critical assertion: the Z-A occurrence's games field must be the
	// complete "{{gameabbrev9|ZA}}", not a truncated "{{gameabbrev9" cut off
	// at the inner template's own "}}". A regex-based non-greedy split gets
	// this wrong silently.
	gamesField, text := findZAOccurrence(t, occs)
	if gamesField != "{{gameabbrev9|ZA}}" {
		t.Fatalf("games field = %q, want exactly %q", gamesField, "{{gameabbrev9|ZA}}")
	}
	wantText := "The user attacks with a strong electric blast. This may also leave targets paralyzed."
	if text != wantText {
		t.Fatalf("text = %q, want %q", text, wantText)
	}
}

func TestParseMoveZADescription_FindsZAEntry(t *testing.T) {
	wikitext := "==Description==\n" + thunderboltDescriptionFixture + "\n==Learnset==\nsomething else\n"
	text, ok := parseMoveZADescription(wikitext)
	if !ok {
		t.Fatal("expected a Z-A description to be found")
	}
	want := "The user attacks with a strong electric blast. This may also leave targets paralyzed."
	if text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
}

func TestParseMoveZADescription_GroupedGameAbbrev_DoesNotMatchZA(t *testing.T) {
	// The LA/SV entry's games field contains gameabbrev8|LA and gameabbrev9|SV
	// but not gameabbrev9|ZA -- must not be mistaken for the Z-A entry, found
	// here by its distinctive text rather than position.
	occs := findTemplates(thunderboltDescriptionFixture, "movedescentry")
	var found bool
	for _, occ := range occs {
		parts := splitTopLevel(occ.Body)
		if len(parts) != 2 || !strings.Contains(parts[1], "The user attacks the target with") {
			continue
		}
		found = true
		codes := gameAbbrevCodes(parts[0])
		for _, c := range codes {
			if c == "ZA" {
				t.Fatalf("LA/SV entry's codes %v unexpectedly contain ZA", codes)
			}
		}
		if len(codes) != 2 {
			t.Fatalf("codes = %v, want 2 (LA, SV)", codes)
		}
	}
	if !found {
		t.Fatal("LA/SV entry not found in fixture")
	}
}

// clefablePokedexFixture is the real, unmodified gen-IX portion of
// Bulbapedia's Clefable (Pokémon) "Pokédex entries" section (fetched
// 2026-08-25). It is the regression fixture for positional form
// attribution: base Clefable and Mega Clefable both have an entry tagged
// exactly "v=Legends: Z-A" with no per-entry form field. The only thing
// that disambiguates them is document position relative to the
// {{Dex/Form|Mega Clefable}} marker.
const clefablePokedexFixture = `{{Dex/Gen/5|gen=IX|reg1=Paldea|reg2=Kitakami|num2=153|reg3=Blueberry|reg4=Lumiose|num4=057|reg5=Hyperspace}}
{{Dex/Entry1|v=Scarlet|t=FFF|entry=Said to live in quiet, remote mountains, this type of fairy has a strong aversion to being seen.}}
{{Dex/Entry1|v=Violet|t=FFF|entry=It has an acute sense of hearing. It can easily hear a pin being dropped nearly 1,100 yards away.}}
{{Dex/Entry1|v=Legends: Z-A|t=FFF|entry=A timid fairy Pokémon that is rarely seen, it will run and hide the moment it senses people.}}
{{Dex/Form|Mega Clefable}}
{{Dex/Entry1|v=Legends: Z-A|t=FFF|entry=It flies by using the power of moonlight to control gravity within a radius of {{tt|over 32 feet|10 meters}} around it.}}
|}
|}`

func clefableIndex() speciesIndex {
	rows := []pokemonRow{
		{ID: 36, Name: "clefable", SpeciesID: 36},
		{ID: 10278, Name: "clefable-mega", SpeciesID: 36},
	}
	return buildSpeciesIndex(rows)
}

func TestParsePokemonDexEntries_FormAttribution_NotByVAlone(t *testing.T) {
	idx := clefableIndex()
	rows, unresolved := parsePokemonDexEntries(clefablePokedexFixture, 36, idx, "Clefable (Pokémon)")
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}

	var baseZA, megaZA *descriptionRow
	for i := range rows {
		r := &rows[i]
		if r.Version != "legends-za" {
			continue
		}
		switch r.ID {
		case 36:
			baseZA = r
		case 10278:
			megaZA = r
		}
	}
	if baseZA == nil {
		t.Fatal("no legends-za row for base Clefable (id 36)")
	}
	if megaZA == nil {
		t.Fatal("no legends-za row for Mega Clefable (id 10278) -- form attribution failed")
	}

	wantBase := "A timid fairy Pokémon that is rarely seen, it will run and hide the moment it senses people."
	wantMega := "It flies by using the power of moonlight to control gravity within a radius of over 32 feet around it."
	if baseZA.Text != wantBase {
		t.Fatalf("base Clefable Z-A text = %q, want %q", baseZA.Text, wantBase)
	}
	if megaZA.Text != wantMega {
		// This is exactly the failure mode a v=-keyed (non-positional) parser
		// produces: it would return wantBase here instead.
		t.Fatalf("Mega Clefable Z-A text = %q, want %q (got the BASE form's text instead: %v)",
			megaZA.Text, wantMega, megaZA.Text == wantBase)
	}

	// Scarlet and Violet entries belong to the base form only -- they appear
	// before the Dex/Form marker.
	for _, r := range rows {
		if r.ID == 10278 && r.Version != "legends-za" {
			t.Fatalf("Mega Clefable unexpectedly has a %s row; Bulbapedia only gives it Z-A text", r.Version)
		}
	}
}

func TestSplitTopLevel_NestedBracesAndBrackets(t *testing.T) {
	body := "{{a|b}}|plain|[[c|d]]"
	got := splitTopLevel(body)
	want := []string{"{{a|b}}", "plain", "[[c|d]]"}
	if len(got) != len(want) {
		t.Fatalf("got %d parts %v, want %d parts %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("part %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCleanWikitext_StripsKnownMarkup(t *testing.T) {
	in := "{{tt|over 32 feet|10 meters}} around <sc>it</sc>.<br>Second line.''emph''"
	got := cleanWikitext(in)
	want := "over 32 feet around it. Second line.emph"
	if got != want {
		t.Fatalf("cleanWikitext(%q) = %q, want %q", in, got, want)
	}
}

func TestPokedexEntriesSection_TolerantOfWikilinkedHeading(t *testing.T) {
	// Greninja (Pokémon) heads its game dex-entries section
	// "===[[Pokédex]] entries===" (a wikilink around "Pokédex") rather than
	// the plain "===Pokédex entries===" every other tested page uses.
	// Verified live: a plain-text-only heading match silently finds nothing
	// on this page, which looks identical to "species has no dex entries."
	wikitext := "===[[Pokédex]] entries===\n{{Dex/Entry1|v=X|entry=test}}\n==Next Section==\nother"
	sec, ok := pokedexEntriesSection(wikitext)
	if !ok {
		t.Fatal("expected the wikilinked heading to be recognized")
	}
	if !strings.Contains(sec, "Dex/Entry1") {
		t.Fatalf("section body = %q, want it to contain the Dex/Entry1 template", sec)
	}
}

func TestParsePokemonDexEntries_KnownGapTaggedNotPlainUnresolved(t *testing.T) {
	// Snorlax's "Mossy" marker is a diagnosed known gap (knowngaps.go):
	// no matching pokemon row exists for it at all. Confirms it comes back
	// tagged Known=true with its reason, not as a plain unresolved item --
	// that tag is what lets report.go collapse the ~38 diagnosed gaps to a
	// one-line count instead of repeating them every run.
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: 143, Name: "snorlax", SpeciesID: 143},
		{ID: 10222, Name: "snorlax-gmax", SpeciesID: 143},
	})
	section := "{{Dex/Form|Mossy}}\n{{Dex/Entry1|v=Scarlet|entry=test}}\n"

	rows, unresolved := parsePokemonDexEntries(section, 143, idx, "Snorlax (Pokémon)")
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows (entries under an unresolved marker are skipped), got %d", len(rows))
	}
	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved item, got %d", len(unresolved))
	}
	if !unresolved[0].Known {
		t.Fatalf("expected the Mossy marker to be tagged Known=true, got %+v", unresolved[0])
	}
	if unresolved[0].Reason == "" {
		t.Fatal("expected a non-empty Reason on a known gap")
	}
}

func TestParsePokemonDexEntries_UnknownGapNotTaggedKnown(t *testing.T) {
	// A marker with no matching candidate AND no knownGaps entry must stay
	// Known=false, so report.go prints it in full -- this is the "new,
	// unexpected problem" path the suppression must never silence.
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: 6, Name: "charizard", SpeciesID: 6},
		{ID: 10034, Name: "charizard-mega-x", SpeciesID: 6},
	})
	section := "{{Dex/Form|Some Brand New Unrecognized Label}}\n{{Dex/Entry1|v=Scarlet|entry=test}}\n"

	_, unresolved := parsePokemonDexEntries(section, 6, idx, "Charizard (Pokémon)")
	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved item, got %d", len(unresolved))
	}
	if unresolved[0].Known {
		t.Fatalf("expected an undiagnosed marker to stay Known=false, got %+v", unresolved[0])
	}
}
