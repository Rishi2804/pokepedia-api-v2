package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPokemonDisplayName(t *testing.T) {
	cases := []struct {
		species, forme, want string
	}{
		{"Gengar", "", "Gengar"},
		{"Mr. Mime", "", "Mr. Mime"},
		{"Charizard", "Mega-X", "Mega Charizard X"},
		{"Mewtwo", "Mega-Y", "Mega Mewtwo Y"},
		{"Venusaur", "Mega", "Mega Venusaur"},
		{"Charizard", "Gmax", "Gigantamax Charizard"},
		{"Urshifu", "Rapid-Strike-Gmax", "Gigantamax Urshifu (Rapid Strike)"},
		{"Groudon", "Primal", "Primal Groudon"},
		{"Raichu", "Alola", "Alolan Raichu"},
		{"Darmanitan", "Galar-Zen", "Galarian Darmanitan (Zen)"},
		{"Typhlosion", "Hisui", "Hisuian Typhlosion"},
		{"Tauros", "Paldea-Combat", "Paldean Tauros (Combat)"},
		{"Necrozma", "Dusk-Mane", "Necrozma (Dusk Mane)"},
		{"Deoxys", "Attack", "Deoxys (Attack)"},
		{"Zygarde", "10%", "Zygarde (10%)"},
		{"Meowstic", "F", "Meowstic (Female)"},
		{"Meowstic", "F-Mega", "Mega Meowstic (Female)"},
		{"Meowstic", "M-Mega", "Mega Meowstic (Male)"},
		{"Tatsugiri", "Droopy-Mega", "Mega Tatsugiri (Droopy)"},
	}
	for _, c := range cases {
		if got := pokemonDisplayName(c.species, c.forme); got != c.want {
			t.Errorf("pokemonDisplayName(%q, %q) = %q, want %q", c.species, c.forme, got, c.want)
		}
	}
}

func TestFallbackPokemonName(t *testing.T) {
	if got := fallbackPokemonName("Alcremie", "alcremie", "alcremie-ruby-swirl"); got != "Alcremie (Ruby Swirl)" {
		t.Errorf("got %q", got)
	}
	if got := fallbackPokemonName("Pikachu", "pikachu", "pikachu"); got != "Pikachu" {
		t.Errorf("got %q, want the species name when the slug adds nothing", got)
	}
}

func TestShowdownSpeciesID(t *testing.T) {
	cases := map[string]string{
		"raichu-alola":       "raichualola", // normalises on its own
		"mr-mime":            "mrmime",
		"deoxys-normal":      "deoxys",           // override
		"meowstic-female":    "meowsticf",        // override
		"meowstic-male-mega": "meowsticmmega",    // override added for the Z-A Megas
		"necrozma-dusk":      "necrozmaduskmane", // override
	}
	for slug, want := range cases {
		if got := showdownSpeciesID(slug); got != want {
			t.Errorf("showdownSpeciesID(%q) = %q, want %q", slug, got, want)
		}
	}
}

// Every override has to point at a species Showdown actually has, or a pokemon
// row would silently get no showdown_id.
func TestShowdownSpeciesOverridesExistInDump(t *testing.T) {
	dex, err := loadShowdownDex()
	if err != nil {
		t.Fatal(err)
	}
	if len(showdownSpeciesOverrides) != 48 {
		t.Errorf("%d overrides, want the 46 inherited from the UI and battle server plus 2 Meowstic Megas", len(showdownSpeciesOverrides))
	}
	for slug, id := range showdownSpeciesOverrides {
		if _, ok := dex.Species[id]; !ok {
			t.Errorf("override %q -> %q: no such Showdown species", slug, id)
		}
	}
	for slug, id := range showdownMoveOverrides {
		if _, ok := dex.Moves[id]; !ok {
			t.Errorf("move override %q -> %q: no such Showdown move", slug, id)
		}
	}
}

func TestShowdownDexLoadsAndResolvesBattleOnlyFormes(t *testing.T) {
	dex, err := loadShowdownDex()
	if err != nil {
		t.Fatal(err)
	}
	mega := dex.Species["charizardmegax"]
	if mega.BattleOnlyBaseID == nil || *mega.BattleOnlyBaseID != "charizard" || len(mega.RequiredItemIDs) == 0 || mega.RequiredItemIDs[0] != "charizarditex" {
		t.Errorf("charizardmegax = %+v", mega)
	}
	if g := dex.Species["charizardgmax"]; !g.Gigantamax || g.BattleOnlyBaseID == nil || *g.BattleOnlyBaseID != "charizard" {
		t.Errorf("charizardgmax = %+v", g)
	}
	if _, ok := dex.Moves["uturn"]; !ok {
		t.Error("uturn missing from the moves dump")
	}
	if _, ok := dex.Abilities["intimidate"]; !ok {
		t.Error("intimidate missing from the abilities dump")
	}
}

func TestEmitNamesMigration(t *testing.T) {
	gengar, base, item := "gengar", int32(6), int32(699)
	name := "Charizard-Mega-X"
	data := &nameData{
		Species: []speciesNameRow{{ID: 122, DisplayName: "Mr. Mime"}},
		Pokemon: []pokemonNameRow{
			{ID: 94, DisplayName: "Gengar", ShowdownID: &gengar, ShowdownName: strPtr("Gengar")},
			{ID: 10034, DisplayName: "Mega Charizard X", ShowdownID: strPtr("charizardmegax"), ShowdownName: &name, BattleOnlyBaseID: &base, RequiredItemID: &item},
		},
		Moves:     []nameRow{{ID: 369, DisplayName: "U-turn", ShowdownID: strPtr("uturn")}},
		Abilities: []nameRow{{ID: 303, DisplayName: "Embody Aspect"}},
	}
	path := filepath.Join(t.TempDir(), "000024_names_data.up.sql")
	if err := emitNamesMigration(path, data); err != nil {
		t.Fatal(err)
	}
	up, _ := os.ReadFile(path)
	for _, want := range []string{
		"(122, 'Mr. Mime')",
		"(94, 'Gengar', 'gengar', 'Gengar', NULL::integer, NULL::integer, false)",
		"(10034, 'Mega Charizard X', 'charizardmegax', 'Charizard-Mega-X', 6, 699, false)",
		"(369, 'U-turn', 'uturn')",
		"(303, 'Embody Aspect', NULL::text)",
		"ALTER TABLE public.species ALTER COLUMN display_name SET NOT NULL;",
		"ADD CONSTRAINT pokemon_showdown_id_key UNIQUE (showdown_id);",
		"REFERENCES public.item (id);",
	} {
		if !strings.Contains(string(up), want) {
			t.Errorf("up missing %q\n%s", want, up)
		}
	}
	down, _ := os.ReadFile(strings.TrimSuffix(path, ".up.sql") + ".down.sql")
	if !strings.Contains(string(down), "DROP CONSTRAINT IF EXISTS pokemon_required_item_id_fkey") {
		t.Errorf("down = %s", down)
	}
}

func strPtr(s string) *string { return &s }
