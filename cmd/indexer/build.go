package main

import (
	"strings"

	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

// nonPokemonPopularity is the rank_feature value for moves and abilities —
// see mapping.json's popularity field.
const nonPokemonPopularity = 8

// indexDoc is the full document written to Elasticsearch (mapping.json).
// It's a superset of search.Doc: aliases/description/effect/popularity feed
// scoring but are excluded from _source.includes on read, so the read-side
// type never needs them.
type indexDoc struct {
	Type        string         `json:"type"`
	EntityID    int32          `json:"entity_id"`
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Aliases     []string       `json:"aliases,omitempty"`
	Description []string       `json:"description,omitempty"`
	Effect      string         `json:"effect,omitempty"`
	Gen         int32          `json:"gen"`
	Popularity  int32          `json:"popularity"`
	Meta        map[string]any `json:"meta,omitempty"`
}

// aliasOverrides covers cases where the natural alias uses different words
// than the slug, not just a reordering of the same ones ("mr-mime" ->
// "mister mime"). It deliberately does not include an un-reordered form of
// the slug itself — the slug field's own tokenization already covers that,
// since the standard tokenizer splits on hyphens the same way it splits on
// spaces, so "raichu-alola" and "Raichu Alola" analyze identically.
var aliasOverrides = map[string][]string{
	"nidoran-f": {"nidoran female"},
	"nidoran-m": {"nidoran male"},
	"mr-mime":   {"mister mime"},
	"mr-rime":   {"mister rime"},
}

func buildAliases(slug, speciesName string) []string {
	var aliases []string
	if speciesName != "" {
		base := util.FormatName(speciesName, true)
		if !strings.EqualFold(base, util.FormatName(slug, true)) {
			aliases = append(aliases, base)
		}
	}
	return append(aliases, aliasOverrides[slug]...)
}

func buildPokemonDoc(r pokemonRow, speciesName string, descriptions map[int32][]string) indexDoc {
	meta := map[string]any{
		"type1":      r.Type1,
		"bst":        r.BST,
		"hp":         r.HP,
		"atk":        r.Atk,
		"def":        r.Def,
		"spatk":      r.SpAtk,
		"spdef":      r.SpDef,
		"speed":      r.Speed,
		"height":     r.Height,
		"weight":     r.Weight,
		"species_id": r.SpeciesID,
		// species.id doubles as the national dex number in this schema —
		// see GetDexNational in db/queries/pokedex.sql, which returns
		// species_id with no separate number column.
		"dex_number": r.SpeciesID,
	}
	if r.Type2 != nil {
		meta["type2"] = *r.Type2
	}

	return indexDoc{
		Type:        "pokemon",
		EntityID:    r.ID,
		Slug:        r.Slug,
		Name:        util.FormatName(r.Slug, true),
		Aliases:     buildAliases(r.Slug, speciesName),
		Description: descriptions[r.ID],
		Gen:         r.Gen,
		Popularity:  r.Popularity,
		Meta:        meta,
	}
}

func buildMoveDoc(r moveRow, descriptions map[int32][]string) indexDoc {
	meta := map[string]any{
		"move_type":  r.Type,
		"move_class": r.Class,
	}
	if r.Power != nil {
		meta["power"] = *r.Power
	}
	if r.Accuracy != nil {
		meta["accuracy"] = *r.Accuracy
	}
	if r.PP != nil {
		meta["pp"] = *r.PP
	}

	var effect string
	if r.Effect != nil {
		effect = *r.Effect
	}

	return indexDoc{
		Type:        "move",
		EntityID:    r.ID,
		Slug:        r.Slug,
		Name:        util.FormatName(r.Slug, false),
		Description: descriptions[r.ID],
		Effect:      effect,
		Gen:         r.Gen,
		Popularity:  nonPokemonPopularity,
		Meta:        meta,
	}
}

func buildAbilityDoc(r abilityRow, descriptions map[int32][]string) indexDoc {
	var effect string
	if r.Effect != nil {
		effect = *r.Effect
	}

	return indexDoc{
		Type:        "ability",
		EntityID:    r.ID,
		Slug:        r.Slug,
		Name:        util.FormatName(r.Slug, false),
		Description: descriptions[r.ID],
		Effect:      effect,
		Gen:         r.Gen,
		Popularity:  nonPokemonPopularity,
	}
}
