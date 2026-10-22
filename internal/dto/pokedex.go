package dto

type PokedexRegionGroup struct {
	Name    string         `json:"name"`
	Pokemon []PokedexEntry `json:"pokemon"`
}

type PokedexEntry struct {
	DexNumber int32   `json:"dexNumber"`
	SpeciesID int32   `json:"speciesId"`
	PokemonID int32   `json:"pokemonId"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Gen       int32   `json:"gen"`
	Type1     string  `json:"type1"`
	Type2     *string `json:"type2"`
}

type TeamBuildingGroup struct {
	ListName string                 `json:"listName"`
	Pokemon  []TeamCandidateSummary `json:"pokemon"`
}

// TeamCandidateSummary is what the list endpoint returns: enough to render and
// filter a candidate card, and nothing else. Abilities, moves and base stats are
// deliberately absent — the team builder's set editor works on one Pokemon at a
// time and fetches them from the single-candidate endpoint, so putting them here
// meant serialising every learnable move for every Pokemon in the version group.
//
// ShowdownID and ShowdownName are Showdown's id ("charizardmegax") and species
// name ("Charizard-Mega-X") for the pokemon, which paste export and the battle
// server use. BattleOnly is set for the formes only a battle reaches.
type TeamCandidateSummary struct {
	ID           int32       `json:"id"`
	Name         string      `json:"name"`
	Slug         string      `json:"slug"`
	ShowdownID   string      `json:"showdownId"`
	ShowdownName string      `json:"showdownName"`
	Gen          int32       `json:"gen"`
	Type1        string      `json:"type1"`
	Type2        *string     `json:"type2"`
	GenderRate   int32       `json:"genderRate"`
	BattleOnly   *BattleOnly `json:"battleOnly"`
}

// BattleOnly describes a forme a team can't submit directly (Mega, Primal,
// Crowned, Gmax, ...): BaseID is the pokemon a team submits for it, and
// RequiredItem the item that reaches it. Either can be absent (Ogerpon's
// masks have an item and no base; a Gmax forme has a base and no item).
type BattleOnly struct {
	BaseID       *int32        `json:"baseId"`
	RequiredItem *RequiredItem `json:"requiredItem"`
	Gigantamax   bool          `json:"gigantamax"`
}

type RequiredItem struct {
	ShowdownID string `json:"showdownId"`
	Name       string `json:"name"`
}

// TeamCandidate is the single-candidate shape. The embedded summary flattens into
// the same JSON object, so the wire format stays one flat record.
type TeamCandidate struct {
	TeamCandidateSummary
	Stats     Stats              `json:"stats"`
	Abilities []CandidateAbility `json:"abilities"`
	Moves     []CandidateMove    `json:"moves"`
}

// Power, Accuracy, PP and Cooldown are the values for the requested game (null
// when the move has none, e.g. status moves or Z-A's cooldown-instead-of-PP).
type CandidateMove struct {
	ID         int32  `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	ShowdownID string `json:"showdownId"`
	Type       string `json:"type"`
	MoveClass  string `json:"moveClass"`
	Power      *int32 `json:"power"`
	Accuracy   *int32 `json:"accuracy"`
	PP         *int32 `json:"pp"`
	Cooldown   *int32 `json:"cooldown"`
	// Newest description within the requested game(s); empty when none exists.
	Description string `json:"description,omitempty"`
}

type CandidateAbility struct {
	ID         int32   `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	ShowdownID *string `json:"showdownId"`
}
