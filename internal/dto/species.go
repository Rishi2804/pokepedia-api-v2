package dto

type SpeciesDetail struct {
	ID      int32            `json:"id"`
	Name    string           `json:"name"`
	Pokemon []PokemonSummary `json:"pokemon"`
}

type PokemonSummary struct {
	ID    int32   `json:"id"`
	Name  string  `json:"name"`
	Gen   int32   `json:"gen"`
	Type1 string  `json:"type1"`
	Type2 *string `json:"type2"`
}
