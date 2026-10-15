package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// choiceScarfJSON is a trimmed PokeAPI /item/ response.
const choiceScarfJSON = `{
  "id": 287, "name": "choice-scarf", "cost": 200, "fling_power": 10, "fling_effect": null,
  "attributes": [{"name": "holdable"}, {"name": "holdable-active"}],
  "category": {"name": "choice", "url": "https://pokeapi.co/api/v2/item-category/32/"},
  "names": [{"name": "Choice Scarf", "language": {"name": "en"}}, {"name": "Mouchoir Choix", "language": {"name": "fr"}}],
  "sprites": {"default": "https://example.test/choice-scarf.png"},
  "effect_entries": [{"effect": "Holder's Speed is\n  increased.", "short_effect": "Boosts Speed\n by 50%.", "language": {"name": "en"}}],
  "flavor_text_entries": [
    {"text": "An item to be held\nby a Pokémon.", "language": {"name": "en"}, "version_group": {"name": "diamond-pearl"}},
    {"text": "Le texte", "language": {"name": "fr"}, "version_group": {"name": "diamond-pearl"}},
    {"text": "Base text.", "language": {"name": "en"}, "version_group": {"name": "scarlet-violet"}},
    {"text": "DLC text.", "language": {"name": "en"}, "version_group": {"name": "the-teal-mask"}},
    {"text": "DLC only.", "language": {"name": "en"}, "version_group": {"name": "the-isle-of-armor"}},
    {"text": "Skipped.", "language": {"name": "en"}, "version_group": {"name": "xd"}}
  ],
  "game_indices": [
    {"generation": {"name": "generation-vi"}}, {"generation": {"name": "generation-iv"}}, {"generation": {"name": "generation-ix"}}
  ],
  "held_by_pokemon": [
    {"pokemon": {"name": "a", "url": "https://pokeapi.co/api/v2/pokemon/10/"},
     "version_details": [{"rarity": 5, "version": {"name": "diamond"}}, {"rarity": 50, "version": {"name": "xd"}}]},
    {"pokemon": {"name": "b", "url": "https://pokeapi.co/api/v2/pokemon/99999/"},
     "version_details": [{"rarity": 5, "version": {"name": "diamond"}}]}
  ]
}`

func decodeItem(t *testing.T, body string) pokeAPIItemResp {
	t.Helper()
	var r pokeAPIItemResp
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestToShowdownID(t *testing.T) {
	cases := map[string]string{
		"Choice Scarf":   "choicescarf",
		"King's Rock":    "kingsrock",
		"Never-Melt Ice": "nevermeltice",
		"Poké Ball":      "pokeball",
		"choice-scarf":   "choicescarf",
	}
	for in, want := range cases {
		if got := toShowdownID(in); got != want {
			t.Errorf("toShowdownID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildItemRow(t *testing.T) {
	resp := decodeItem(t, choiceScarfJSON)
	row := buildItemRow(resp, "battle", map[int32]bool{10: true})

	if row.Gen == nil || *row.Gen != 4 {
		t.Errorf("gen = %v, want earliest generation 4", row.Gen)
	}
	if row.FlingPower == nil || *row.FlingPower != 10 || row.FlingEffect != nil {
		t.Errorf("fling = %v / %v", row.FlingPower, row.FlingEffect)
	}
	if row.Effect == nil || *row.Effect != "Holder's Speed is increased." {
		t.Errorf("effect = %v", row.Effect)
	}
	if row.ShortEffect == nil || *row.ShortEffect != "Boosts Speed by 50%." {
		t.Errorf("short effect = %v", row.ShortEffect)
	}
	if row.Sprite == nil || *row.Sprite != "items/choice-scarf.png" {
		t.Errorf("sprite = %v", row.Sprite)
	}
	if row.Pocket != "battle" || row.Category != "choice" || row.Cost != 200 {
		t.Errorf("row = %+v", row)
	}
}

func TestBuildItemRowGenFallsBackToFlavorText(t *testing.T) {
	resp := decodeItem(t, `{"id": 2233, "name": "clefablite", "category": {"name": "mega-stones"},
	  "flavor_text_entries": [{"text": "x", "language": {"name": "en"}, "version_group": {"name": "legends-z-a"}}]}`)
	row := buildItemRow(resp, "misc", nil)
	if row.Gen == nil || *row.Gen != 9 {
		t.Errorf("gen = %v, want 9", row.Gen)
	}
}

func TestItemDescriptions(t *testing.T) {
	rows := itemDescriptions(decodeItem(t, choiceScarfJSON))
	got := map[string]string{}
	for _, r := range rows {
		got[r.Version] = r.Text
	}
	want := map[string]string{
		"diamond": "An item to be held by a Pokémon.",
		"pearl":   "An item to be held by a Pokémon.",
		"scarlet": "Base text.", // base game wins over the Teal Mask copy
		"violet":  "Base text.",
		"sword":   "DLC only.", // DLC fills a gap the base groups left
		"shield":  "DLC only.",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d games %v, want %d", len(got), got, len(want))
	}
	for g, text := range want {
		if got[g] != text {
			t.Errorf("%s = %q, want %q", g, got[g], text)
		}
	}
}

func TestItemHeldBy(t *testing.T) {
	rows := itemHeldBy(decodeItem(t, choiceScarfJSON), map[int32]bool{10: true})
	if len(rows) != 1 {
		t.Fatalf("got %+v, want only pokemon 10 in diamond (XD and unknown pokemon dropped)", rows)
	}
	if r := rows[0]; r.ItemID != 287 || r.PokemonID != 10 || r.Version != "diamond" || r.Rarity != 5 {
		t.Errorf("row = %+v", r)
	}
}

func TestMatchShowdownItem(t *testing.T) {
	showdown := map[string]showdownItem{
		"choicescarf": {ID: "choicescarf", Name: "Choice Scarf", LegalFormats: []string{"4"}},
		"kingsrock":   {ID: "kingsrock", Name: "King's Rock"},
	}
	if sd, ok := matchShowdownItem(decodeItem(t, choiceScarfJSON), showdown); !ok || sd.ID != "choicescarf" {
		t.Errorf("by name: %v %v", sd, ok)
	}
	bySlug := pokeAPIItemResp{Name: "kings-rock"}
	if sd, ok := matchShowdownItem(bySlug, showdown); !ok || sd.ID != "kingsrock" {
		t.Errorf("by slug: %v %v", sd, ok)
	}
	if _, ok := matchShowdownItem(pokeAPIItemResp{Name: "potion"}, showdown); ok {
		t.Error("potion should not match")
	}
}

func TestIsHoldableItem(t *testing.T) {
	cases := []struct {
		slug, category string
		want           bool
	}{
		{"leftovers", "held-items", true},
		{"rocky-helmet", "held-items", true},
		{"abomasite", "mega-stones", true},
		{"firium-z--held", "z-crystals", true},
		{"ultranecrozium-z--held", "unused", true},
		{"red-orb", "plot-advancement", true},
		{"sitrus-berry", "medicine", true},
		{"fire-stone", "evolution", false},
		{"helix-fossil", "dex-completion", false},
		{"great-ball", "standard-balls", false},
		{"razz-berry", "baking-only", false},
		{"firium-z--bag", "unused", false},
		{"potion", "healing", true}, // category is necessary, not sufficient: Showdown must also know it
	}
	for _, c := range cases {
		if got := isHoldableItem(c.slug, c.category); got != c.want {
			t.Errorf("isHoldableItem(%q, %q) = %v, want %v", c.slug, c.category, got, c.want)
		}
	}
}

func TestShowdownItemsEmbedded(t *testing.T) {
	m, err := loadShowdownItems()
	if err != nil {
		t.Fatal(err)
	}
	ls, ok := m["leftovers"]
	if !ok || len(ls.LegalFormats) == 0 {
		t.Fatalf("leftovers = %+v", ls)
	}
	if _, ok := m["potion"]; ok {
		t.Error("potion must not be in the Showdown dump")
	}
}

func TestMachineID(t *testing.T) {
	for _, id := range []string{"tr00", "tr99", "tm01"} {
		if !machineID.MatchString(id) {
			t.Errorf("%q should be a machine id", id)
		}
	}
	for _, id := range []string{"trace", "leftovers", "tr"} {
		if machineID.MatchString(id) {
			t.Errorf("%q should not be a machine id", id)
		}
	}
}

func TestIDFromPokeAPIURL(t *testing.T) {
	if id, ok := idFromPokeAPIURL("https://pokeapi.co/api/v2/item/287/"); !ok || id != 287 {
		t.Errorf("got %d %v", id, ok)
	}
	if _, ok := idFromPokeAPIURL("https://pokeapi.co/api/v2/item/"); ok {
		t.Error("non-numeric tail should fail")
	}
}

func TestEmitItemMigration(t *testing.T) {
	scarf := "choicescarf"
	gen := int32(4)
	rows := []itemRow{{
		ID: 287, Name: "choice-scarf", Category: "choice", Pocket: "battle", Gen: &gen,
		Cost: 200, ShowdownID: &scarf, Holdable: true, LegalFormats: []string{"4", "nationaldex"},
		Descriptions: []descriptionRow{{Entity: entityItem, ID: 287, Version: "diamond", Text: "It's held."}},
		HeldBy:       []itemHeldByRow{{ItemID: 287, PokemonID: 10, Version: "diamond", Rarity: 5}},
	}}
	path := filepath.Join(t.TempDir(), "000014_items_data.up.sql")
	if err := emitItemMigration(path, rows); err != nil {
		t.Fatal(err)
	}
	up, _ := os.ReadFile(path)
	for _, want := range []string{
		"(287, 'choice-scarf', 'choice', 'battle', 4, 200, NULL, NULL, NULL, NULL, NULL, 'choicescarf', true, ARRAY['4','nationaldex']::text[])",
		"(287, 'diamond', 'It''s held.')",
		"(287, 10, 'diamond', 5)",
		"ON CONFLICT (id) DO NOTHING",
	} {
		if !strings.Contains(string(up), want) {
			t.Errorf("up missing %q\n%s", want, up)
		}
	}
	down, _ := os.ReadFile(strings.TrimSuffix(path, ".up.sql") + ".down.sql")
	if !strings.Contains(string(down), "DELETE FROM public.item WHERE id = ANY(ARRAY[287])") {
		t.Errorf("down = %s", down)
	}
}
