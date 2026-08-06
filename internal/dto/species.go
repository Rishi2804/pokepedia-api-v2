package dto

type SpeciesDetail struct {
	ID      int32           `json:"id"`
	Name    string          `json:"name"`
	Pokemon []PokemonDetail `json:"pokemon"`
}
