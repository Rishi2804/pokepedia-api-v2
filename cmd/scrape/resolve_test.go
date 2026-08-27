package main

import (
	"strings"
	"testing"
)

// buildIndex is a small helper for form-resolution tests: species is the
// base pokemon row (id == species id) and forms are its siblings, mirroring
// the shape fetchPokemonRows/buildSpeciesIndex produce from the DB.
func buildIndex(species pokemonRow, forms ...pokemonRow) speciesIndex {
	all := append([]pokemonRow{species}, forms...)
	return buildSpeciesIndex(all)
}

func TestResolveFormMarker_MegaXYDisambiguation(t *testing.T) {
	// Real Bulbapedia markers from Charizard (Pokémon), fetched 2026-08-25:
	// {{Dex/Form|Mega Charizard X}}, {{Dex/Form|Mega Charizard Y}}. Both
	// share the "mega" token, so a plain any-overlap match would tie; only
	// scoring by overlap SIZE (mega+x beats mega alone) disambiguates them.
	idx := buildIndex(
		pokemonRow{ID: 6, Name: "charizard", SpeciesID: 6},
		pokemonRow{ID: 10034, Name: "charizard-mega-x", SpeciesID: 6},
		pokemonRow{ID: 10035, Name: "charizard-mega-y", SpeciesID: 6},
	)

	cases := []struct {
		marker string
		want   int32
	}{
		{"Mega Charizard X", 10034},
		{"Mega Charizard Y", 10035},
	}
	for _, c := range cases {
		id, ok := resolveFormMarker(c.marker, 6, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", c.marker)
		}
		if id != c.want {
			t.Fatalf("resolveFormMarker(%q) = %d, want %d", c.marker, id, c.want)
		}
	}
}

func TestResolveFormMarker_RegionalFormGenericMarker(t *testing.T) {
	// Real markers from Meowth (Pokémon): {{Dex/Form|Alolan Form}},
	// {{Dex/Form|Galarian Form}}, {{Dex/Form|Gigantamax}} -- none of these
	// name the species, unlike Mega markers.
	idx := buildIndex(
		pokemonRow{ID: 52, Name: "meowth", SpeciesID: 52},
		pokemonRow{ID: 10160, Name: "meowth-alola", SpeciesID: 52},
		pokemonRow{ID: 10161, Name: "meowth-galar", SpeciesID: 52},
		pokemonRow{ID: 10197, Name: "meowth-gmax", SpeciesID: 52},
	)

	cases := []struct {
		marker string
		want   int32
	}{
		{"Alolan Form", 10160},
		{"Galarian Form", 10161},
		{"Gigantamax", 10197},
	}
	for _, c := range cases {
		id, ok := resolveFormMarker(c.marker, 52, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", c.marker)
		}
		if id != c.want {
			t.Fatalf("resolveFormMarker(%q) = %d, want %d", c.marker, id, c.want)
		}
	}
}

func TestResolveFormMarker_ZygardePercentForms(t *testing.T) {
	// Real markers from Zygarde (Pokémon): {{Dex/Form|10% Forme}},
	// {{Dex/Form|50% Forme}}, {{Dex/Form|Complete Forme}},
	// {{Dex/Form|Mega Zygarde}}.
	idx := buildIndex(
		pokemonRow{ID: 718, Name: "zygarde", SpeciesID: 718},
		pokemonRow{ID: 10118, Name: "zygarde-10", SpeciesID: 718},
		pokemonRow{ID: 10119, Name: "zygarde-50", SpeciesID: 718},
		pokemonRow{ID: 10120, Name: "zygarde-complete", SpeciesID: 718},
		pokemonRow{ID: 10301, Name: "zygarde-mega", SpeciesID: 718},
	)

	cases := []struct {
		marker string
		want   int32
	}{
		{"10% Forme", 10118},
		{"50% Forme", 10119},
		{"Complete Forme", 10120},
		{"Mega Zygarde", 10301},
	}
	for _, c := range cases {
		id, ok := resolveFormMarker(c.marker, 718, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", c.marker)
		}
		if id != c.want {
			t.Fatalf("resolveFormMarker(%q) = %d, want %d", c.marker, id, c.want)
		}
	}
}

func TestResolveFormMarker_RotomAppliances_ReversedWordOrder(t *testing.T) {
	// Real markers from Rotom (Pokémon): "Heat Rotom", "Wash Rotom", etc. --
	// the adjective precedes the species name, the OPPOSITE order from this
	// project's own util.FormatName ("Rotom Heat"), which is why the
	// resolver cannot rely on FormatName's output as the marker text.
	idx := buildIndex(
		pokemonRow{ID: 479, Name: "rotom", SpeciesID: 479},
		pokemonRow{ID: 10008, Name: "rotom-heat", SpeciesID: 479},
		pokemonRow{ID: 10009, Name: "rotom-wash", SpeciesID: 479},
		pokemonRow{ID: 10010, Name: "rotom-frost", SpeciesID: 479},
		pokemonRow{ID: 10011, Name: "rotom-fan", SpeciesID: 479},
		pokemonRow{ID: 10012, Name: "rotom-mow", SpeciesID: 479},
	)

	cases := []struct {
		marker string
		want   int32
	}{
		{"Heat Rotom", 10008},
		{"Wash Rotom", 10009},
		{"Frost Rotom", 10010},
		{"Fan Rotom", 10011},
		{"Mow Rotom", 10012},
		{"Rotom", 479}, // resets to base, as seen between generation blocks
	}
	for _, c := range cases {
		id, ok := resolveFormMarker(c.marker, 479, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", c.marker)
		}
		if id != c.want {
			t.Fatalf("resolveFormMarker(%q) = %d, want %d", c.marker, id, c.want)
		}
	}
}

func TestResolveFormMarker_NecrozmaFusions_SpeciesNameOmitted(t *testing.T) {
	// Real markers from Necrozma (Pokémon): "Dusk Mane", "Dawn Wings",
	// "Ultra Necrozma" -- the first two omit the species name entirely, and
	// "Mane"/"Wings" are noise tokens with no DB counterpart; the match must
	// still succeed on the "Dusk"/"Dawn" overlap alone.
	idx := buildIndex(
		pokemonRow{ID: 800, Name: "necrozma", SpeciesID: 800},
		pokemonRow{ID: 10155, Name: "necrozma-dusk", SpeciesID: 800},
		pokemonRow{ID: 10156, Name: "necrozma-dawn", SpeciesID: 800},
		pokemonRow{ID: 10157, Name: "necrozma-ultra", SpeciesID: 800},
	)

	cases := []struct {
		marker string
		want   int32
	}{
		{"Dusk Mane", 10155},
		{"Dawn Wings", 10156},
		{"Ultra Necrozma", 10157},
	}
	for _, c := range cases {
		id, ok := resolveFormMarker(c.marker, 800, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", c.marker)
		}
		if id != c.want {
			t.Fatalf("resolveFormMarker(%q) = %d, want %d", c.marker, id, c.want)
		}
	}
}

func TestResolveFormMarker_Unmatched_ReportsRatherThanGuesses(t *testing.T) {
	idx := buildIndex(
		pokemonRow{ID: 6, Name: "charizard", SpeciesID: 6},
		pokemonRow{ID: 10034, Name: "charizard-mega-x", SpeciesID: 6},
	)
	if _, ok := resolveFormMarker("Some Unrecognized Label", 6, idx); ok {
		t.Fatal("expected no match for a marker with zero token overlap, got a match")
	}
}

func TestSuffixTokens_SetDifferenceNotPositional(t *testing.T) {
	form := pokemonRow{ID: 10034, Name: "charizard-mega-x", SpeciesID: 6}
	rootWords := tokenSet(strings.Split("charizard", "-"))
	sfx := suffixTokens(form, rootWords)
	if len(sfx) != 2 || !sfx["mega"] || !sfx["x"] {
		t.Fatalf("suffixTokens = %v, want {mega, x}", sfx)
	}
}

func TestResolveFormMarker_NoLiteralBaseRow(t *testing.T) {
	// Real markers from Morpeko (Pokémon): {{Dex/Form|Full Belly Mode}},
	// {{Dex/Form|Hangry Mode}}. Morpeko has NO row named plain "morpeko" --
	// only morpeko-full-belly and morpeko-hangry -- so idx.root must recover
	// "morpeko" from the siblings' common prefix rather than requiring a
	// literal id == species_id row.
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: 10230, Name: "morpeko-full-belly", SpeciesID: 877},
		{ID: 10231, Name: "morpeko-hangry", SpeciesID: 877},
	})

	cases := []struct {
		marker string
		want   int32
	}{
		{"Full Belly Mode", 10230},
		{"Hangry Mode", 10231},
	}
	for _, c := range cases {
		id, ok := resolveFormMarker(c.marker, 877, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", c.marker)
		}
		if id != c.want {
			t.Fatalf("resolveFormMarker(%q) = %d, want %d", c.marker, id, c.want)
		}
	}
}

func TestResolveFormMarker_SingleRowSpecies_AnyMarkerResolves(t *testing.T) {
	// Silvally, Unown, Sinistea, and Polteageist each have exactly one
	// pokemon row despite Bulbapedia narrating several named states
	// ("Type: Normal" / "''All other forms''" for Silvally's 18 memory
	// types; "Phony Form" / "Antique Form" for Sinistea) that this database
	// never split into separate rows -- the sole row is the only possible
	// target regardless of the marker's wording.
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: 773, Name: "silvally", SpeciesID: 773},
	})

	for _, marker := range []string{"Type: Normal", "''All other forms''"} {
		id, ok := resolveFormMarker(marker, 773, idx)
		if !ok {
			t.Fatalf("resolveFormMarker(%q): no match", marker)
		}
		if id != 773 {
			t.Fatalf("resolveFormMarker(%q) = %d, want 773", marker, id)
		}
	}
}

func TestResolveFormMarker_MultiRowSpecies_UnmatchedMarkerStillUnresolved(t *testing.T) {
	// Guards against over-generalising the single-row fallback: Sinistea
	// (1 row) and a hypothetical 2-row species must behave differently.
	// Here "Antique Form" matches nothing and the species has 2 rows, so it
	// must stay unresolved rather than being force-fit onto either one.
	idx := buildIndex(
		pokemonRow{ID: 854, Name: "sinistea", SpeciesID: 854},
		pokemonRow{ID: 10300, Name: "sinistea-fake", SpeciesID: 854}, // hypothetical
	)
	if _, ok := resolveFormMarker("Antique Form", 854, idx); ok {
		t.Fatal("expected no match: marker overlaps neither candidate's suffix")
	}
}

func TestResolveFormMarker_KnownBaseLabel(t *testing.T) {
	// Real markers from Zacian (Pokémon): {{Dex/Form|Hero of Many Battles}},
	// {{Dex/Form|Crowned Sword}}. "Hero of Many Battles" shares no token
	// with zacian-crowned's suffix {crowned}, so it needs the explicit
	// knownBaseLabels table; "Crowned Sword" already resolves by ordinary
	// token overlap and must keep doing so.
	idx := buildIndex(
		pokemonRow{ID: 888, Name: "zacian", SpeciesID: 888},
		pokemonRow{ID: 10250, Name: "zacian-crowned", SpeciesID: 888},
	)

	id, ok := resolveFormMarker("Hero of Many Battles", 888, idx)
	if !ok || id != 888 {
		t.Fatalf("resolveFormMarker(Hero of Many Battles) = (%d, %v), want (888, true)", id, ok)
	}
	id, ok = resolveFormMarker("Crowned Sword", 888, idx)
	if !ok || id != 10250 {
		t.Fatalf("resolveFormMarker(Crowned Sword) = (%d, %v), want (10250, true)", id, ok)
	}
}

func TestBuildTitle_NormalizesCurlyApostrophe(t *testing.T) {
	// PokeAPI's English name for move 717 is "Nature’s Madness" (curly
	// right single quote); Bulbapedia's real page title uses a straight
	// apostrophe. Verified live -- without this, the request 404s and looks
	// like a missing page rather than an encoding mismatch.
	got := buildTitle("Nature’s Madness", "move")
	want := "Nature's Madness (move)"
	if got != want {
		t.Fatalf("buildTitle = %q, want %q", got, want)
	}
}

func TestResolveFormMarker_ExtraTokenTiebreak(t *testing.T) {
	// Real markers, all previously tied at matched=1 for two candidates
	// before the extra-token tiebreak: "Mega Garchomp" scores equally
	// against garchomp-mega ({mega}) and garchomp-mega-z ({mega, z}) by
	// overlap count alone. Preferring the candidate with fewer suffix
	// tokens the marker never mentioned (extra=0 beats extra=1) picks the
	// one that doesn't require the marker to have silently omitted "Z".
	idx := buildIndex(
		pokemonRow{ID: 445, Name: "garchomp", SpeciesID: 445},
		pokemonRow{ID: 10058, Name: "garchomp-mega", SpeciesID: 445},
		pokemonRow{ID: 10309, Name: "garchomp-mega-z", SpeciesID: 445},
	)
	id, ok := resolveFormMarker("Mega Garchomp", 445, idx)
	if !ok || id != 10058 {
		t.Fatalf("resolveFormMarker(Mega Garchomp) = (%d, %v), want (10058, true)", id, ok)
	}

	// Tatsugiri: "Droopy Form" must prefer tatsugiri-droopy over
	// tatsugiri-droopy-mega the same way.
	tIdx := buildIndex(
		pokemonRow{ID: 978, Name: "tatsugiri", SpeciesID: 978},
		pokemonRow{ID: 10259, Name: "tatsugiri-droopy", SpeciesID: 978},
		pokemonRow{ID: 10323, Name: "tatsugiri-droopy-mega", SpeciesID: 978},
	)
	id, ok = resolveFormMarker("Droopy Form", 978, tIdx)
	if !ok || id != 10259 {
		t.Fatalf("resolveFormMarker(Droopy Form) = (%d, %v), want (10259, true)", id, ok)
	}
}

func TestResolveFormMarker_ExtraTokenTiebreak_GenuineTieStillUnresolved(t *testing.T) {
	// Real markers from Ogerpon (Pokémon): "Teal Mask" (handled separately
	// via knownBaseLabels) aside, a hypothetical marker overlapping equally
	// with two DIFFERENT-suffix candidates at equal extra must still stay
	// unresolved -- the tiebreak breaks near-ties, not genuine ones.
	idx := buildIndex(
		pokemonRow{ID: 1017, Name: "ogerpon", SpeciesID: 1017},
		pokemonRow{ID: 10270, Name: "ogerpon-wellspring-mask", SpeciesID: 1017},
		pokemonRow{ID: 10271, Name: "ogerpon-hearthflame-mask", SpeciesID: 1017},
	)
	// "Mask" alone overlaps both candidates' suffix by exactly 1, with
	// exactly 1 extra token each (wellspring/hearthflame respectively) --
	// a genuine, unbreakable tie.
	if _, ok := resolveFormMarker("Mask", 1017, idx); ok {
		t.Fatal("expected no match: marker ties equally between two real candidates")
	}
}

func TestResolveFormMarker_ApostropheStripped(t *testing.T) {
	// Real marker from Oricorio (Pokémon): {{Dex/Form|Pa'u Style}}. Without
	// stripping the apostrophe, tokenizing splits "Pa'u" into the two
	// meaningless fragments "pa" and "u", neither of which matches
	// oricorio-pau's suffix "pau" -- the DB slug itself drops the
	// apostrophe entirely, same as farfetchd for Farfetch'd.
	idx := buildIndex(
		pokemonRow{ID: 741, Name: "oricorio-baile", SpeciesID: 741},
		pokemonRow{ID: 10173, Name: "oricorio-pau", SpeciesID: 741},
		pokemonRow{ID: 10174, Name: "oricorio-sensu", SpeciesID: 741},
	)
	id, ok := resolveFormMarker("Pa'u Style", 741, idx)
	if !ok {
		t.Fatal("resolveFormMarker(Pa'u Style): no match")
	}
	if id != 10173 {
		t.Fatalf("resolveFormMarker(Pa'u Style) = %d, want 10173", id)
	}
}

func TestResolveFormMarker_KnownBaseLabels_NewEntries(t *testing.T) {
	// Each pair is real: (species with a literal base, marker that names
	// that base using vocabulary absent from any sibling's suffix).
	cases := []struct {
		name   string
		base   pokemonRow
		alt    pokemonRow
		marker string
	}{
		{"Palafin", pokemonRow{964, "palafin", 964}, pokemonRow{10276, "palafin-hero", 964}, "Zero Form"},
		{"Hoopa", pokemonRow{720, "hoopa", 720}, pokemonRow{10087, "hoopa-unbound", 720}, "Hoopa Confined"},
		{"Gimmighoul", pokemonRow{999, "gimmighoul", 999}, pokemonRow{10263, "gimmighoul-roaming", 999}, "Chest Form"},
		{"Terapagos", pokemonRow{1024, "terapagos", 1024}, pokemonRow{10276, "terapagos-terastal", 1024}, "Normal Form"},
		{"Castform", pokemonRow{351, "castform", 351}, pokemonRow{10013, "castform-sunny", 351}, "Normal"},
		{"Oinkologne", pokemonRow{916, "oinkologne", 916}, pokemonRow{10262, "oinkologne-female", 916}, "Male"},
	}
	for _, c := range cases {
		idx := buildIndex(c.base, c.alt)
		id, ok := resolveFormMarker(c.marker, c.base.SpeciesID, idx)
		if !ok || id != c.base.ID {
			t.Fatalf("%s: resolveFormMarker(%q) = (%d, %v), want (%d, true)", c.name, c.marker, id, ok, c.base.ID)
		}
	}
}
