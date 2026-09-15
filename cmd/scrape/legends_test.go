package main

import (
	"strings"
	"testing"
)

// Fixtures below are real, unmodified Bulbapedia wikitext (fetched
// 2026-08-25/27, cached under cmd/scrape/.cache/bulbapedia), used to
// validate legends.go's field parsers and section walker against ground
// truth rather than invented text.

// cyndaquilLALevelFixture is Cyndaquil (Pokemon)/Generation VIII learnset's
// full "By leveling up" section: mainline level8 entries (must be ignored --
// wrong template name), then the {{gameabbrev8|LA}}-marked block of 7
// {{learnlist/levelLA|...}} entries this test cares about, single form (no
// form heading at all).
const cyndaquilLALevelFixture = `====By [[Level|leveling up]]====
{{gameabbrev8|BDSP}}
{{learnlist/levelh/8|Cyndaquil|Fire|Fire|2}}
{{learnlist/level8|1|Tackle|Normal|Physical|40|100|35||}}
{{learnlist/level8|1|Leer|Normal|Status|&mdash;|100|30||}}
{{learnlist/levelf/8|Cyndaquil|Fire|Fire|2}}

{{gameabbrev8|LA}}
{{learnlist/levelh/LA|Cyndaquil|Fire|Fire|2}}
{{learnlist/levelLA|1|10|Quick Attack|Normal|Physical|40|50|30|100|100|20||}}
{{learnlist/levelLA|6|15|Ember|Fire|Special|40|50|30|100|100|25||'''}}
{{learnlist/levelLA|11|20|Rollout|Rock|Physical|40|50|30|90|100|20||}}
{{learnlist/levelLA|18|28|Flame Wheel|Fire|Physical|60|75|45|100|100|20||'''}}
{{learnlist/levelLA|25|35|Swift|Normal|Special|60|75|45|&mdash;|&mdash;|20||}}
{{learnlist/levelLA|34|45|Flamethrower|Fire|Special|80|100|60|100|100|10||'''}}
{{learnlist/levelLA|43|54|Overheat|Fire|Special|110|130|90|90|100|5||'''}}
{{learnlist/levelf/8|Cyndaquil|Fire|Fire|2}}

====By [[TM]]====
`

// cyndaquilLATutorFixture is the same page's "By tutoring" section: 6
// {{learnlist/tutorPLA|...}} entries, including Rest (every stat dashed --
// a Status move with no power/accuracy of its own) and Aerial Ace (dashed
// accuracy only).
const cyndaquilLATutorFixture = `====By [[Move Tutor|tutoring]]====
{{gameabbrev8|LA}}
{{learnlist/tutorh/PLA|Cyndaquil|Fire|Fire|2}}
{{learnlist/tutorPLA|Aerial Ace|Flying|Physical|60|75|45|&mdash;|&mdash;|20||}}
{{learnlist/tutorPLA|Flamethrower|Fire|Special|80|100|60|100|100|10||'''}}
{{learnlist/tutorPLA|Iron Tail|Steel|Physical|100|120|80|75|90|5||}}
{{learnlist/tutorPLA|Rest|Psychic|Status|&mdash;|&mdash;|&mdash;|&mdash;|&mdash;|10||}}
{{learnlist/tutorPLA|Swift|Normal|Special|60|75|45|&mdash;|&mdash;|20||}}
{{learnlist/tutorPLA|Wild Charge|Electric|Physical|85|105|65|100|100|10||}}
{{learnlist/tutorf/8|Cyndaquil|Fire|Fire|2}}

[[it:Cyndaquil/Mosse apprese in ottava generazione]]
`

// phioneLATutorMalformedFixture is Phione (Pokemon)/Generation VIII
// learnset's "By tutoring" section -- the one entry cmd/scrape's plan
// flagged in advance as malformed on live Bulbapedia: Zen Headbutt's fields
// are shifted one position left (missing PowerAgile), leaving PP's slot
// blank instead of "10". Every other entry on the page (and every other
// tutorPLA entry across the whole cache using this move) has the correct
// 9-field layout, confirmed by cross-checking every "Zen Headbutt" tutorPLA
// occurrence in the cache during development.
const phioneLATutorMalformedFixture = `====By [[Move Tutor|tutoring]]====
{{gameabbrev8|LA}}
{{learnlist/tutorh/PLA|Phione|Water|Water|4}}
{{learnlist/tutorPLA|Bubble|Water|Special|40|50|30|100|100|25||'''}}
{{learnlist/tutorPLA|Zen Headbutt|Psychic|Physical|80|90|60|100|10||}}
{{learnlist/tutorf/8|Phione|Water|Water|4}}
`

// gyaradosLALevelFixture is Gyarados (Pokemon)/Generation VIII learnset's
// LA level-up block, chosen for two edge cases neither Cyndaquil nor
// Phantump above exercises: Splash's Mastery field is the literal word
// "None" (no Strong/Agile Style benefit at all, every power/accuracy field
// dashed), and Water Pulse's Learn field is a nested {{tt|Evo.|...}}
// tooltip (learned automatically on evolving into Gyarados) instead of a
// plain number.
const gyaradosLALevelFixture = `====By [[Level|leveling up]]====
{{gameabbrev8|LA}}
{{learnlist/levelh/LA|Gyarados|Water|Flying|1}}
{{learnlist/levelLA|1|None|Splash|Normal|Status|—|—|—|—|—|40||}}
{{learnlist/levelLA|{{tt|Evo.|Learned upon evolving}}|18|Water Pulse|Water|Special|60|75|45|—|—|20||'''}}
{{learnlist/levelLA|5|14|Twister|Dragon|Special|40|50|30|100|100|25||}}
{{learnlist/levelf/8|Gyarados|Water|Flying|1}}

====By [[TM]]/[[TR]]====
`

// basculinLALevelFixture is Basculin (Pokemon)/Generation VIII learnset's
// full "By leveling up" section: two form headings, only the second of
// which ("White-Striped Basculin") has any levelLA entries at all --
// "Red-Striped/Blue-Striped Basculin"'s entries are all mainline level8,
// which must never be picked up by a levelLA-only walk regardless of which
// form is "current" at that point.
const basculinLALevelFixture = `====By [[Level|leveling up]]====
=====Red-Striped/Blue-Striped Basculin=====
{{gameabbrev8|SwSh}}
{{learnlist/levelh/8|Basculin|Water|Water|5}}
{{learnlist/level8|1|Water Gun|Water|Special|40|100|25||'''}}
{{learnlist/level8|1|Tail Whip|Normal|Status|—|100|30}}
{{learnlist/levelf/8|Basculin|Water|Water|5}}

=====White-Striped Basculin=====
{{gameabbrev8|LA}}
{{learnlist/levelh/LA|Basculin|Water|Water|5}}
{{learnlist/levelLA|1|10|Tackle|Normal|Physical|40|50|30|100|100|30||}}
{{learnlist/levelLA|6|15|Aqua Jet|Water|Physical|40|50|30|100|100|20||'''}}
{{learnlist/levelLA|11|20|Bite|Dark|Physical|60|75|45|100|100|20||}}
{{learnlist/levelf/8|Basculin|Water|Water|5}}
`

// steelixLALevelFixture is Steelix (Pokemon)/Generation VIII learnset's
// full "By leveling up" section -- the one page in the whole cache that
// divides this section by GAME NAME rather than by form ("Pokemon Sword,
// Shield, Brilliant Diamond, and Shining Pearl" / "Pokemon Legends:
// Arceus"), even though Steelix has no alternate form relevant to either
// game. Without the breedingFormOverrides entries for these two exact
// headings, the second heading fails to resolve (no species form matches
// its text) and every levelLA entry under it is silently dropped.
const steelixLALevelFixture = `====By [[Level|leveling up]]====
=====Pokémon Sword, Shield, Brilliant Diamond, and Shining Pearl=====
{{learnlist/levelh/8|Steelix|Steel|Ground|2}}
{{learnlist/level8|1|Crunch|Dark|Physical|80|100|15}}
{{learnlist/levelf/8|Steelix|Steel|Ground|2}}

=====Pokémon Legends: Arceus=====
{{learnlist/levelh/LA|Steelix|Steel|Ground|2}}
{{learnlist/levelLA|1|10|Rollout|Rock|Physical|40|50|30|90|100|20||}}
{{learnlist/levelLA|6|15|Tackle|Normal|Physical|40|50|30|100|100|30||}}
{{learnlist/levelf/8|Steelix|Steel|Ground|2}}
`

// floetteZALevelFixture is Floette (Pokemon)'s full "By leveling up"
// section: an "All regular forms" heading (Floette's five flower colors
// have no separate pokemon row -- see knowngaps.go -- so this covers all of
// them via the single "floette" row) followed by its own SV block and then
// its {{gameabbrev9|ZA}}-marked Legends: Z-A block, then a second heading
// ("Eternal Flower Floette") whose entries must never be attributed to the
// base row. Also exercises an {{tt|Evo.|...}} Learn field on levelZA
// (Moonblast), the same pattern gyaradosLALevelFixture exercises for LA.
const floetteZALevelFixture = `====By [[Level|leveling up]]====
=====All regular forms=====
{{gameabbrev9|SV}}
{{learnlist/levelh/9|Floette|Fairy|Fairy|6}}
{{learnlist/level9|1|Tackle|Normal|Physical|40|100|35}}
{{learnlist/levelf/9|Floette|Fairy|Fairy|6}}

{{gameabbrev9|ZA}}
{{learnlist/levelh/ZA|Floette|Fairy|Fairy|6}}
{{learnlist/levelZA|{{tt|Evo.|Learned upon evolving}}|22|Moonblast|Fairy|Special|95|8||'''}}
{{learnlist/levelZA|1|10|Vine Whip|Grass|Physical|45|6}}
{{learnlist/levelZA|1|10|Tackle|Normal|Physical|40|4}}
{{learnlist/levelf/9|Floette|Fairy|Fairy|6}}

=====Eternal Flower Floette=====
{{learnlist/levelh/9|Floette|Fairy|Fairy|6}}
{{learnlist/level9|1|Tackle|Normal|Physical|40|100|35}}
{{learnlist/levelf/9|Floette|Fairy|Fairy|6}}
`

// phantumpZALevelFixture and phantumpZATMFixture are Phantump (Pokemon)'s
// full "By leveling up" and "By TM" sections: single form, no game-marker
// ambiguity, exercising the ordinary levelZA/tmZA path end to end (mainline
// SV entries alongside the ZA-only block that must be the only thing that
// ends up in the parsed rows).
const phantumpZALevelFixture = `====By [[Level|leveling up]]====
{{gameabbrev9|SV}}
{{learnlist/levelh/9|Phantump|Ghost|Grass|6}}
{{learnlist/level9|1|Astonish|Ghost|Physical|30|100|15||'''}}
{{learnlist/level9|1|Tackle|Normal|Physical|40|100|35||}}
{{learnlist/levelf/9|Phantump|Ghost|Grass|6}}

{{gameabbrev9|ZA}}
{{learnlist/levelh/ZA|Phantump|Ghost|Grass|6}}
{{learnlist/levelZA|1|10|Tackle|Normal|Physical|40|4}}
{{learnlist/levelZA|8|11|Leech Seed|Grass|Status|—|12}}
{{learnlist/levelZA|12|15|Confuse Ray|Ghost|Status|—|6}}
{{learnlist/levelf/9|Phantump|Ghost|Grass|6}}

====By [[TM]]====
`

const phantumpZATMFixture = `====By [[TM]]====
{{gameabbrev9|SV}}
{{learnlist/tmh/9|Phantump|Ghost|Grass|6}}
{{learnlist/tm9|TM007|Protect|Normal|Status|—|—|10||}}
{{learnlist/tmf/9|Phantump|Ghost|Grass|6}}

{{gameabbrev9|ZA}}
{{learnlist/tmh/ZA|Phantump|Ghost|Grass|6}}
{{learnlist/tmZA|TM012|Rock Slide|Rock|Physical|75|7}}
{{learnlist/tmZA|TM017|Protect|Normal|Status|—|15}}
{{learnlist/tmf/9|Phantump|Ghost|Grass|6}}

====By {{pkmn|breeding}}====
`

// shayminLALevelFixture is Shaymin (Pokemon)/Generation VIII learnset's
// full "By leveling up" section: a mainline BDSP block with its own
// "Land Forme"/"Sky Forme" headings and footers, followed immediately by
// an LA block with NO heading of its own at all. This is the exact shape
// that exposed a real bug: without tracking {{gameabbrev8|...}} markers
// and the footers that close each heading's block, the LA entries here
// silently inherited "Sky Forme" -- the last heading seen, but for an
// entirely different, already-closed mainline block -- attributing every
// LA move to shaymin-sky alone. PokeAPI confirms both formes have real,
// different Legends: Arceus movesets (22 vs 14 moves), so that's not
// incomplete, it's wrong.
const shayminLALevelFixture = `====By [[Level|leveling up]]====
{{gameabbrev8|BDSP}}
=====Land Forme=====
{{learnlist/levelh/8|Shaymin|Grass|Grass|4}}
{{learnlist/level8|1|Growth|Normal|Status|—|—|20||}}
{{learnlist/levelf/8|Shaymin|Grass|Grass|4}}

=====Sky Forme=====
{{learnlist/levelh/8|Shaymin|Grass|Flying|4}}
{{learnlist/level8|1|Growth|Normal|Status|—|—|20||}}
{{learnlist/levelf/8|Shaymin|Grass|Flying|4}}

{{gameabbrev8|LA}}
{{learnlist/levelh/LA|Shaymin|Grass|Grass|4}}
{{learnlist/levelLA|1|12|Leafage|Grass|Physical|40|50|30|100|100|25||}}
{{learnlist/levelLA|6|17|Quick Attack|Normal|Physical|40|50|30|100|100|20||}}
{{learnlist/levelf/8|Shaymin|Grass|Grass|4|form=yes}}
`

func TestParseLegendsSection_Shaymin_MarkerResetsStaleHeading(t *testing.T) {
	const shayminID = 492
	// No literal base and no legendsSpeciesDefaults entry for Shaymin --
	// its LA tutor section genuinely can't be attributed (see
	// knowngaps.go), and its level-up section relies entirely on the
	// "Land Forme"/"Sky Forme" headings, never on a species-wide default.
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: shayminID, Name: "shaymin-land", SpeciesID: shayminID},
		{ID: 10006, Name: "shaymin-sky", SpeciesID: shayminID},
	})
	moveIndex := map[string]int32{normalizeMoveKey("Leafage"): 1, normalizeMoveKey("Quick Attack"): 2}

	sec, ok := methodSection(shayminLALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("level section not found")
	}
	rows, unresolved := parseLegendsSection(sec, shayminID, idx, "Shaymin (Pokémon)/Generation VIII learnset", moveIndex, "learnlist/levelLA", "legends-arceus", parseLevelLAFields)
	// Neither Land nor Sky Forme is a resolvable target for these untagged
	// LA entries (no heading applies -- "Sky Forme" already closed), so
	// this must report unresolved, not silently attribute to shaymin-sky.
	if len(rows) != 0 {
		t.Fatalf(`got %d rows, want 0 -- LA entries must not silently inherit the stale "Sky Forme" heading from the closed mainline BDSP block: %+v`, len(rows), rows)
	}
	if len(unresolved) != 1 {
		t.Fatalf("unresolved = %v, want exactly 1 (the unattributable LA block)", unresolved)
	}
}

func TestParseLevelLAFields(t *testing.T) {
	entries := findTemplates(cyndaquilLALevelFixture, "learnlist/levelLA")
	if len(entries) != 7 {
		t.Fatalf("got %d levelLA entries, want 7", len(entries))
	}
	// Quick Attack: {{learnlist/levelLA|1|10|Quick Attack|Normal|Physical|40|50|30|100|100|20||}}
	fields, ok := parseLevelLAFields(splitTopLevel(entries[0].Body))
	if !ok {
		t.Fatal("parseLevelLAFields(Quick Attack) = false, want true")
	}
	if fields.MoveName != "Quick Attack" || fields.Level != 1 || *fields.SecondLevel != 10 {
		t.Fatalf("Quick Attack fields = %+v", fields)
	}
	if *fields.PowerBase != 40 || *fields.PowerStrong != 50 || *fields.PowerAgile != 30 {
		t.Fatalf("Quick Attack power = %+v", fields)
	}
	if *fields.Accuracy1 != 100 || *fields.Accuracy2 != 100 || *fields.PP != 20 {
		t.Fatalf("Quick Attack accuracy/pp = %+v", fields)
	}

	// Swift: dashed accuracy (&mdash;) on both accuracy fields.
	fields, ok = parseLevelLAFields(splitTopLevel(entries[4].Body))
	if !ok || fields.MoveName != "Swift" {
		t.Fatalf("parseLevelLAFields(Swift) = %+v, %v", fields, ok)
	}
	if fields.Accuracy1 != nil || fields.Accuracy2 != nil {
		t.Fatalf("Swift's dashed accuracy fields should be nil, got %+v", fields)
	}
}

func TestParseLevelLAFields_NoneMasteryAndEvoLearn(t *testing.T) {
	entries := findTemplates(gyaradosLALevelFixture, "learnlist/levelLA")
	if len(entries) != 3 {
		t.Fatalf("got %d levelLA entries, want 3", len(entries))
	}

	splash, ok := parseLevelLAFields(splitTopLevel(entries[0].Body))
	if !ok || splash.MoveName != "Splash" {
		t.Fatalf("parseLevelLAFields(Splash) = %+v, %v", splash, ok)
	}
	if splash.Level != 1 {
		t.Fatalf("Splash Level = %d, want 1", splash.Level)
	}
	if splash.SecondLevel != nil {
		t.Fatalf(`Splash Mastery = %v, want nil (literal "None")`, *splash.SecondLevel)
	}
	if splash.PowerBase != nil || splash.Accuracy1 != nil {
		t.Fatalf("Splash's all-dashed stat fields should be nil, got %+v", splash)
	}
	if splash.PP == nil || *splash.PP != 40 {
		t.Fatalf("Splash PP = %v, want 40", splash.PP)
	}

	waterPulse, ok := parseLevelLAFields(splitTopLevel(entries[1].Body))
	if !ok || waterPulse.MoveName != "Water Pulse" {
		t.Fatalf("parseLevelLAFields(Water Pulse) = %+v, %v", waterPulse, ok)
	}
	if waterPulse.Level != 0 {
		t.Fatalf(`Water Pulse Level = %d, want 0 (learned on evolving -- {{tt|Evo.|...}} Learn field)`, waterPulse.Level)
	}
	if waterPulse.SecondLevel == nil || *waterPulse.SecondLevel != 18 {
		t.Fatalf("Water Pulse Mastery = %v, want 18", waterPulse.SecondLevel)
	}
}

func TestParseTutorPLAFields(t *testing.T) {
	entries := findTemplates(cyndaquilLATutorFixture, "learnlist/tutorPLA")
	if len(entries) != 6 {
		t.Fatalf("got %d tutorPLA entries, want 6", len(entries))
	}

	aerialAce, ok := parseTutorPLAFields(splitTopLevel(entries[0].Body))
	if !ok || aerialAce.MoveName != "Aerial Ace" || aerialAce.Method != "tutor" || aerialAce.Level != 0 {
		t.Fatalf("parseTutorPLAFields(Aerial Ace) = %+v, %v", aerialAce, ok)
	}
	if aerialAce.Accuracy1 != nil || aerialAce.Accuracy2 != nil {
		t.Fatalf("Aerial Ace's dashed accuracy should be nil, got %+v", aerialAce)
	}

	rest, ok := parseTutorPLAFields(splitTopLevel(entries[3].Body))
	if !ok || rest.MoveName != "Rest" {
		t.Fatalf("parseTutorPLAFields(Rest) = %+v, %v", rest, ok)
	}
	if rest.PowerBase != nil || rest.Accuracy1 != nil || rest.PP == nil || *rest.PP != 10 {
		t.Fatalf("Rest fields = %+v", rest)
	}
}

func TestParseTutorPLAFields_MalformedPhione(t *testing.T) {
	entries := findTemplates(phioneLATutorMalformedFixture, "learnlist/tutorPLA")
	if len(entries) != 2 {
		t.Fatalf("got %d tutorPLA entries, want 2", len(entries))
	}
	// Bubble is a normal, correctly-shaped entry.
	if _, ok := parseTutorPLAFields(splitTopLevel(entries[0].Body)); !ok {
		t.Fatal("parseTutorPLAFields(Bubble) = false, want true (well-formed entry)")
	}
	// Zen Headbutt is shifted one field left on live Bulbapedia -- PowerAgile
	// is missing, so what should be PP's slot is empty. This must be
	// reported as malformed, never silently stored with wrong values.
	if _, ok := parseTutorPLAFields(splitTopLevel(entries[1].Body)); ok {
		t.Fatal("parseTutorPLAFields(Phione's malformed Zen Headbutt) = true, want false")
	}
}

func TestParseLevelZAFields(t *testing.T) {
	entries := findTemplates(phantumpZALevelFixture, "learnlist/levelZA")
	if len(entries) != 3 {
		t.Fatalf("got %d levelZA entries, want 3", len(entries))
	}
	tackle, ok := parseLevelZAFields(splitTopLevel(entries[0].Body))
	if !ok || tackle.MoveName != "Tackle" || tackle.Method != "level-up" {
		t.Fatalf("parseLevelZAFields(Tackle) = %+v, %v", tackle, ok)
	}
	if tackle.Level != 1 || *tackle.SecondLevel != 10 {
		t.Fatalf("Tackle Learn/Plus = %+v", tackle)
	}
	if *tackle.PowerBase != 40 || *tackle.Cooldown != 4 {
		t.Fatalf("Tackle power/cooldown = %+v", tackle)
	}
	if tackle.PowerStrong != nil || tackle.Accuracy1 != nil || tackle.PP != nil {
		t.Fatalf("levelZA has no Strong/Agile power, no accuracy, no PP -- got %+v", tackle)
	}

	leechSeed, ok := parseLevelZAFields(splitTopLevel(entries[1].Body))
	if !ok || leechSeed.PowerBase != nil {
		t.Fatalf("Leech Seed (Status, dashed power) = %+v, %v", leechSeed, ok)
	}
}

func TestParseLevelZAFields_EvoLearn(t *testing.T) {
	entries := findTemplates(floetteZALevelFixture, "learnlist/levelZA")
	if len(entries) != 3 {
		t.Fatalf("got %d levelZA entries, want 3", len(entries))
	}
	moonblast, ok := parseLevelZAFields(splitTopLevel(entries[0].Body))
	if !ok || moonblast.MoveName != "Moonblast" {
		t.Fatalf("parseLevelZAFields(Moonblast) = %+v, %v", moonblast, ok)
	}
	if moonblast.Level != 0 {
		t.Fatalf(`Moonblast Level = %d, want 0 (learned on evolving into Floette)`, moonblast.Level)
	}
	if moonblast.SecondLevel == nil || *moonblast.SecondLevel != 22 {
		t.Fatalf("Moonblast Plus = %v, want 22", moonblast.SecondLevel)
	}
}

func TestParseTmZAFields(t *testing.T) {
	entries := findTemplates(phantumpZATMFixture, "learnlist/tmZA")
	if len(entries) != 2 {
		t.Fatalf("got %d tmZA entries, want 2", len(entries))
	}
	rockSlide, ok := parseTmZAFields(splitTopLevel(entries[0].Body))
	if !ok || rockSlide.MoveName != "Rock Slide" || rockSlide.Method != "machine" || rockSlide.Level != 0 {
		t.Fatalf("parseTmZAFields(Rock Slide) = %+v, %v", rockSlide, ok)
	}
	if *rockSlide.PowerBase != 75 || *rockSlide.Cooldown != 7 {
		t.Fatalf("Rock Slide power/cooldown = %+v", rockSlide)
	}

	protect, ok := parseTmZAFields(splitTopLevel(entries[1].Body))
	if !ok || protect.PowerBase != nil {
		t.Fatalf("Protect (Status, dashed power) = %+v, %v", protect, ok)
	}
}

func TestMethodSection_BoundsToNextHeading(t *testing.T) {
	sec, ok := methodSection(cyndaquilLALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("methodSection(levelUpHeadingRe) not found")
	}
	if strings.Contains(sec, "By [[TM]]") {
		t.Fatal("methodSection body leaked past the next level-4 heading")
	}
	if !strings.Contains(sec, "learnlist/levelLA") {
		t.Fatal("methodSection body is missing the LA block it should contain")
	}

	sec, ok = methodSection(cyndaquilLATutorFixture, tutoringHeadingRe)
	if !ok {
		t.Fatal("methodSection(tutoringHeadingRe) not found")
	}
	if !strings.Contains(sec, "learnlist/tutorPLA") {
		t.Fatal("methodSection body is missing the tutor block it should contain")
	}
}

func TestParseLegendsSection_Cyndaquil_SingleForm(t *testing.T) {
	const cyndaquilID = 155
	idx := buildSpeciesIndex([]pokemonRow{{ID: cyndaquilID, Name: "cyndaquil", SpeciesID: cyndaquilID}})
	moveIndex := map[string]int32{}
	for i, n := range []string{"Quick Attack", "Ember", "Rollout", "Flame Wheel", "Swift", "Flamethrower", "Overheat",
		"Aerial Ace", "Iron Tail", "Rest", "Wild Charge"} {
		moveIndex[normalizeMoveKey(n)] = int32(i + 1)
	}

	levelSec, ok := methodSection(cyndaquilLALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("level section not found")
	}
	levelRows, unresolved := parseLegendsSection(levelSec, cyndaquilID, idx, "Cyndaquil (Pokémon)/Generation VIII learnset", moveIndex, "learnlist/levelLA", "legends-arceus", parseLevelLAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if len(levelRows) != 7 {
		t.Fatalf("got %d level-up rows, want 7", len(levelRows))
	}
	for _, r := range levelRows {
		if r.PokemonID != cyndaquilID || r.Version != "legends-arceus" || r.Method != "level-up" {
			t.Fatalf("row %+v: unexpected pokemon_id/version/method", r)
		}
	}

	tutorSec, ok := methodSection(cyndaquilLATutorFixture, tutoringHeadingRe)
	if !ok {
		t.Fatal("tutor section not found")
	}
	tutorRows, unresolved := parseLegendsSection(tutorSec, cyndaquilID, idx, "Cyndaquil (Pokémon)/Generation VIII learnset", moveIndex, "learnlist/tutorPLA", "legends-arceus", parseTutorPLAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if len(tutorRows) != 6 {
		t.Fatalf("got %d tutor rows, want 6", len(tutorRows))
	}
	for _, r := range tutorRows {
		if r.Method != "tutor" || r.Level != 0 {
			t.Fatalf("row %+v: want method=tutor level=0", r)
		}
	}
}

func TestParseLegendsSection_MalformedEntryReportedNotStored(t *testing.T) {
	const phioneID = 489
	idx := buildSpeciesIndex([]pokemonRow{{ID: phioneID, Name: "phione", SpeciesID: phioneID}})
	moveIndex := map[string]int32{normalizeMoveKey("Bubble"): 1, normalizeMoveKey("Zen Headbutt"): 2}

	sec, ok := methodSection(phioneLATutorMalformedFixture, tutoringHeadingRe)
	if !ok {
		t.Fatal("tutor section not found")
	}
	rows, unresolved := parseLegendsSection(sec, phioneID, idx, "Phione (Pokémon)/Generation VIII learnset", moveIndex, "learnlist/tutorPLA", "legends-arceus", parseTutorPLAFields)
	if len(rows) != 1 || rows[0].MoveID != 1 {
		t.Fatalf("got rows %+v, want exactly the well-formed Bubble row", rows)
	}
	if len(unresolved) != 1 {
		t.Fatalf("unresolved = %v, want exactly 1 (the malformed Zen Headbutt entry)", unresolved)
	}
}

func TestParseLegendsSection_Basculin_OnlyWhiteStripedGetsLARows(t *testing.T) {
	const (
		speciesID = 550
		redID     = 550
		blueID    = 10016
		whiteID   = 10247
	)
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: redID, Name: "basculin-red-striped", SpeciesID: speciesID},
		{ID: blueID, Name: "basculin-blue-striped", SpeciesID: speciesID},
		{ID: whiteID, Name: "basculin-white-striped", SpeciesID: speciesID},
	})
	moveIndex := map[string]int32{}
	for i, n := range []string{"Water Gun", "Tail Whip", "Tackle", "Aqua Jet", "Bite"} {
		moveIndex[normalizeMoveKey(n)] = int32(i + 1)
	}

	sec, ok := methodSection(basculinLALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("level section not found")
	}
	rows, unresolved := parseLegendsSection(sec, speciesID, idx, "Basculin (Pokémon)/Generation VIII learnset", moveIndex, "learnlist/levelLA", "legends-arceus", parseLevelLAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (Tackle/Aqua Jet/Bite)", len(rows))
	}
	for _, r := range rows {
		if r.PokemonID != whiteID {
			t.Fatalf("row %+v: pokemon_id = %d, want %d (basculin-white-striped) -- Red/Blue-Striped's own entries are mainline level8, never levelLA", r, r.PokemonID, whiteID)
		}
	}
}

func TestParseLegendsSection_Steelix_GameNameHeadingOverride(t *testing.T) {
	const steelixID = 208
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: steelixID, Name: "steelix", SpeciesID: steelixID},
		{ID: 10072, Name: "steelix-mega", SpeciesID: steelixID},
	})
	moveIndex := map[string]int32{normalizeMoveKey("Crunch"): 1, normalizeMoveKey("Rollout"): 2, normalizeMoveKey("Tackle"): 3}

	sec, ok := methodSection(steelixLALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("level section not found")
	}
	rows, unresolved := parseLegendsSection(sec, steelixID, idx, "Steelix (Pokémon)/Generation VIII learnset", moveIndex, "learnlist/levelLA", "legends-arceus", parseLevelLAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none -- both game-name headings need breedingFormOverrides entries", unresolved)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (Rollout, Tackle) -- without the override these are silently dropped", len(rows))
	}
	for _, r := range rows {
		if r.PokemonID != steelixID {
			t.Fatalf("row %+v: pokemon_id = %d, want %d (steelix, not steelix-mega)", r, r.PokemonID, steelixID)
		}
	}
}

func TestParseLegendsSection_Floette_AllRegularFormsOverride(t *testing.T) {
	const floetteID = 670
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: floetteID, Name: "floette", SpeciesID: floetteID},
		{ID: 10061, Name: "floette-eternal", SpeciesID: floetteID},
		{ID: 10296, Name: "floette-mega", SpeciesID: floetteID},
	})
	moveIndex := map[string]int32{normalizeMoveKey("Moonblast"): 1, normalizeMoveKey("Vine Whip"): 2, normalizeMoveKey("Tackle"): 3}

	sec, ok := methodSection(floetteZALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("level section not found")
	}
	rows, unresolved := parseLegendsSection(sec, floetteID, idx, "Floette (Pokémon)", moveIndex, "learnlist/levelZA", "legends-za", parseLevelZAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	// 3 moves x 2 targets each (floette itself, plus floette-mega via
	// zaMegaByBase[670] -- floettite is a real Legends: Z-A Mega Stone, see
	// that map's comment) = 6 rows, none for floette-eternal.
	if len(rows) != 6 {
		t.Fatalf("got %d rows, want 6 (3 moves x floette + floette-mega)", len(rows))
	}
	for _, r := range rows {
		if r.PokemonID != floetteID && r.PokemonID != 10296 {
			t.Fatalf("row %+v: pokemon_id = %d, want %d (floette) or 10296 (floette-mega) -- must not leak onto floette-eternal (its own separate heading/entries follow)", r, r.PokemonID, floetteID)
		}
	}
}

func TestParseLegendsSection_Phantump_ZALevelAndTM(t *testing.T) {
	const phantumpID = 708
	idx := buildSpeciesIndex([]pokemonRow{{ID: phantumpID, Name: "phantump", SpeciesID: phantumpID}})
	moveIndex := map[string]int32{}
	for i, n := range []string{"Tackle", "Leech Seed", "Confuse Ray", "Protect", "Rock Slide"} {
		moveIndex[normalizeMoveKey(n)] = int32(i + 1)
	}

	levelSec, ok := methodSection(phantumpZALevelFixture, levelUpHeadingRe)
	if !ok {
		t.Fatal("level section not found")
	}
	levelRows, unresolved := parseLegendsSection(levelSec, phantumpID, idx, "Phantump (Pokémon)", moveIndex, "learnlist/levelZA", "legends-za", parseLevelZAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if len(levelRows) != 3 {
		t.Fatalf("got %d level-up rows, want 3 (mainline SV entries must be excluded)", len(levelRows))
	}

	tmSec, ok := methodSection(phantumpZATMFixture, tmHeadingRe)
	if !ok {
		t.Fatal("TM section not found")
	}
	tmRows, unresolved := parseLegendsSection(tmSec, phantumpID, idx, "Phantump (Pokémon)", moveIndex, "learnlist/tmZA", "legends-za", parseTmZAFields)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if len(tmRows) != 2 {
		t.Fatalf("got %d TM rows, want 2", len(tmRows))
	}
	for _, r := range tmRows {
		if r.Method != "machine" {
			t.Fatalf("row %+v: method = %q, want machine", r, r.Method)
		}
	}
}
