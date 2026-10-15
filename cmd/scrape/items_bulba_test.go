package main

import (
	"reflect"
	"testing"
)

// clefabliteWikitext is trimmed from Bulbapedia's Clefablite page.
const clefabliteWikitext = `{{ItemInfobox/head
|name=Clefablite
|gen=IX
|colorscheme=items
}}
The '''Clefablite''' is a type of [[held item]].

===Description===
{{movedesc|items}}
{{movedescentry|{{gameabbrev9|ZA}}|One of a variety of mysterious Mega Stones. A {{p|Clefable}} holding this stone will be able to Mega Evolve during battle.
}}
{{movedescentry|{{gameabbrevss|Champs}}|A held item that allows Clefable to Mega Evolve.}}
|}
`

func TestParseBulbapediaItem(t *testing.T) {
	info := parseBulbapediaItem(clefabliteWikitext)
	if info.Gen == nil || *info.Gen != 9 {
		t.Errorf("gen = %v, want 9", info.Gen)
	}
	want := map[string]string{
		"legends-za": "One of a variety of mysterious Mega Stones. A Clefable holding this stone will be able to Mega Evolve during battle.",
	}
	if !reflect.DeepEqual(info.Descriptions, want) {
		t.Errorf("descriptions = %v, want %v (Champions is ignored)", info.Descriptions, want)
	}
	if len(info.UnknownCodes) != 0 {
		t.Errorf("unknown codes = %v", info.UnknownCodes)
	}
}

func TestParseBulbapediaItemGroupedCodes(t *testing.T) {
	info := parseBulbapediaItem(`{{ItemInfobox/head
|gen=II
}}
{{movedescentry|{{gameabbrev2|GSC}}|Heals 10 HP.}}
{{movedescentry|{{gameabbrev9|SVZA}}<br>{{gameabbrev9|MYSTERY}}|Later text.}}
`)
	for _, g := range []string{"gold", "silver", "crystal"} {
		if info.Descriptions[g] != "Heals 10 HP." {
			t.Errorf("%s = %q", g, info.Descriptions[g])
		}
	}
	for _, g := range []string{"scarlet", "violet", "legends-za"} {
		if info.Descriptions[g] != "Later text." {
			t.Errorf("%s = %q", g, info.Descriptions[g])
		}
	}
	if !reflect.DeepEqual(info.UnknownCodes, []string{"MYSTERY"}) {
		t.Errorf("unknown codes = %v, want [MYSTERY]", info.UnknownCodes)
	}
}

func TestParseBulbapediaItemEmpty(t *testing.T) {
	if !parseBulbapediaItem("A disambiguation page with {{no|templates}} we read.").isEmpty() {
		t.Error("page with no infobox gen or descriptions should be empty")
	}
}

func TestNewBulbapediaOnlyRow(t *testing.T) {
	row := newBulbapediaOnlyRow(20008, showdownItem{ID: "mintberry", Name: "Mint Berry", Gen: 2, LegalFormats: []string{"2", "nationaldex"}})
	if row.Name != "mint-berry" || row.Category != "medicine" || row.Pocket != "berries" {
		t.Errorf("row = %+v", row)
	}
	if row.Sprite == nil || *row.Sprite != "items/mint-berry.png" {
		t.Errorf("sprite = %v", row.Sprite)
	}
	if !row.Holdable || row.ShowdownID == nil || *row.ShowdownID != "mintberry" || row.Gen == nil || *row.Gen != 2 {
		t.Errorf("row = %+v", row)
	}
	if mail := newBulbapediaOnlyRow(20007, showdownItem{ID: "mail", Name: "Mail", Gen: 2}); mail.Pocket != "mail" || mail.Category != "all-mail" {
		t.Errorf("mail = %+v", mail)
	}
	if bow := newBulbapediaOnlyRow(20011, showdownItem{ID: "pinkbow", Name: "Pink Bow", Gen: 2}); bow.Name != "pink-bow" || bow.Pocket != "misc" {
		t.Errorf("bow = %+v", bow)
	}
}

func TestNewBulbapediaOnlyRowSlugOverride(t *testing.T) {
	row := newBulbapediaOnlyRow(20015, showdownItem{ID: "stick", Name: "Stick", Gen: 2})
	if row.Name != "stick-legacy" || row.DisplayName != "Stick" {
		t.Errorf("row = %+v, want slug stick-legacy so it can't collide with PokeAPI's stick (Leek)", row)
	}
}

func TestBulbapediaOnlyIDsAreUniqueAndHigh(t *testing.T) {
	seen := map[int32]string{}
	for sd, id := range bulbapediaOnlyItemIDs {
		if id < 20000 {
			t.Errorf("%s: id %d below 20000 could collide with PokeAPI", sd, id)
		}
		if prev, dup := seen[id]; dup {
			t.Errorf("id %d assigned to both %s and %s", id, prev, sd)
		}
		seen[id] = sd
	}
}
