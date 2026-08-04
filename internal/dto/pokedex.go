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
	ListName string          `json:"listName"`
	Pokemon  []TeamCandidate `json:"pokemon"`
}

type TeamCandidate struct {
	ID         int32              `json:"id"`
	Name       string             `json:"name"`
	Gen        int32              `json:"gen"`
	Type1      string             `json:"type1"`
	Type2      *string            `json:"type2"`
	GenderRate int32              `json:"genderRate"`
	Abilities  []CandidateAbility `json:"abilities"`
	Moves      []CandidateMove    `json:"moves"`
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
