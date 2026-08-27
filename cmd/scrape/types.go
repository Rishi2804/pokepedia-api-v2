package main

import "fmt"

// entity names the table a descriptionRow belongs to: pokemondescriptions,
// movedescriptions, or abilitydescriptions.
type entity string

const (
	entityPokemon entity = "pokemon"
	entityMove    entity = "move"
	entityAbility entity = "ability"
)

// descriptionRow is one (entity_id, version, text) triple destined for
// pokemondescriptions, movedescriptions, or abilitydescriptions. Source and
// SourceTitle are provenance only -- used in the dry-run report and the
// -emit migration header -- never written to the DB itself.
type descriptionRow struct {
	Entity      entity
	ID          int32
	Version     string // public.game enum value
	Text        string
	Source      string // "pokeapi" or "bulbapedia"
	SourceTitle string // Bulbapedia page title, empty for pokeapi rows
}

// unresolvedItem is one thing the scraper could not resolve. Known gaps
// (see knowngaps.go) carry Reason and get collapsed to a one-line summary
// in the report instead of a full detail line every run -- see report.go.
type unresolvedItem struct {
	Message string
	Known   bool
	Reason  string
}

func unresolvedf(format string, a ...any) unresolvedItem {
	return unresolvedItem{Message: fmt.Sprintf(format, a...)}
}
