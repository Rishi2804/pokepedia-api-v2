package dto

type SpeciesDetail struct {
	ID      int32           `json:"id"`
	Slug    string          `json:"slug"`
	Name    string          `json:"name"`
	Pokemon []PokemonDetail `json:"pokemon"`
}
