package dto

// Slug is the URL form (/ability/air-lock) and ShowdownID Showdown's own id;
// Name is the display name, which can't be turned back into either
// ("Mind's Eye" is "minds-eye"). ShowdownID is null for the one PokeAPI
// ability Showdown splits in four (Embody Aspect).
type AbilitySnap struct {
	ID         int32   `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	ShowdownID *string `json:"showdownId"`
	Gen        int32   `json:"gen"`
}

type AbilityDetail struct {
	ID           int32         `json:"id"`
	Slug         string        `json:"slug"`
	Name         string        `json:"name"`
	ShowdownID   *string       `json:"showdownId"`
	Gen          int32         `json:"gen"`
	Effect       string        `json:"effect"`
	Descriptions []Description `json:"descriptions"`
	Pokemon      []PokemonSnap `json:"pokemon"`
}
