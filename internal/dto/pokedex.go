package dto

type PokedexRegionGroup struct {
	Name    string         `json:"name"`
	Pokemon []PokedexEntry `json:"pokemon"`
}

type PokedexEntry struct {
	DexNumber int32   `json:"dexNumber"`
	SpeciesID int32   `json:"speciesId"`
	PokemonID int32   `json:"pokemonId"`
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
type TeamCandidateSummary struct {
	ID         int32   `json:"id"`
	Name       string  `json:"name"`
	Slug       string  `json:"slug"`
	Gen        int32   `json:"gen"`
	Type1      string  `json:"type1"`
	Type2      *string `json:"type2"`
	GenderRate int32   `json:"genderRate"`
}

// TeamCandidate is the single-candidate shape. The embedded summary flattens into
// the same JSON object, so the wire format stays one flat record.
type TeamCandidate struct {
	TeamCandidateSummary
	Stats     Stats              `json:"stats"`
	Abilities []CandidateAbility `json:"abilities"`
	Moves     []CandidateMove    `json:"moves"`
}

type CandidateMove struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	MoveClass string `json:"moveClass"`
}

type CandidateAbility struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}
