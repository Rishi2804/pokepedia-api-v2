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

// moveDetailRow is one (pokemon_id, move_id, method, version) egg-move fact
// destined for movedetails. It is a separate type from descriptionRow --
// not shoehorned into it -- because movedetails has a genuinely different
// shape (no text column, an extra method/level_learned pair) and a
// different write policy (see insertEggMove in queries.go: movedetails has
// no unique constraint, so it can't reuse ON CONFLICT).
type moveDetailRow struct {
	PokemonID   int32
	MoveID      int32
	Version     string // public.group enum value
	Source      string
	SourceTitle string
}

// legendsMoveRow is one (pokemon_id, move_id, method, version) learnset
// fact for Legends: Arceus or Legends: Z-A, plus the per-row stat extras
// public.legendsmovevalues carries (see 000010_legends_move_values.up.sql
// for why these can't live on movedetails itself or be flattened to
// pastmovevalues's per-move-only shape -- Dialga/Palkia/Giratina's Origin
// Formes have genuinely different power for their own signature moves).
// Fields the specific game doesn't use are left nil (Legends: Z-A has no
// mastery/style/second-accuracy/PP; Legends: Arceus has no cooldown).
type legendsMoveRow struct {
	PokemonID int32
	MoveID    int32
	Version   string // "legends-arceus" or "legends-za"
	Method    string // public.learnmethod: "level-up", "tutor", "machine"
	Level     int32  // level_learned; 0 for tutor/machine

	SecondLevel *int32 // LA mastery level / ZA "plus" level
	PowerBase   *int32
	PowerStrong *int32 // LA only
	PowerAgile  *int32 // LA only
	Accuracy1   *int32
	Accuracy2   *int32 // LA only
	PP          *int32 // LA only
	Cooldown    *int32 // ZA only

	Source      string
	SourceTitle string
}
