package dto

type AbilitySnap struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
	Gen  int32  `json:"gen"`
}

type AbilityDetail struct {
	ID           int32         `json:"id"`
	Name         string        `json:"name"`
	Gen          int32         `json:"gen"`
	Effect       string        `json:"effect"`
	Descriptions []Description `json:"descriptions"`
	Pokemon      []PokemonSnap `json:"pokemon"`
}
