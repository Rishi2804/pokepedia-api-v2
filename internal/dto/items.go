package dto

// ItemSnap is one row of the item list. Slug is the URL form
// (/item/choice-scarf); Name is the display name, which can't be turned back
// into the slug ("King's Rock" is "kings-rock").
type ItemSnap struct {
	ID       int32  `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Gen      *int32 `json:"gen"`
	Category string `json:"category"`
	Pocket   string `json:"pocket"`
	Image    string `json:"image"`
}

type ItemDetail struct {
	ID           int32         `json:"id"`
	Slug         string        `json:"slug"`
	Name         string        `json:"name"`
	Gen          *int32        `json:"gen"`
	Category     string        `json:"category"`
	Pocket       string        `json:"pocket"`
	Cost         int32         `json:"cost"`
	FlingPower   *int32        `json:"flingPower"`
	FlingEffect  *string       `json:"flingEffect"`
	Effect       string        `json:"effect"`
	ShortEffect  string        `json:"shortEffect"`
	Image        string        `json:"image"`
	Holdable     bool          `json:"holdable"`
	ShowdownID   *string       `json:"showdownId"`
	LegalFormats []string      `json:"legalFormats"`
	Descriptions []Description `json:"descriptions"`
	// Pokemon that can be found holding the item in the wild.
	Pokemon []PokemonSnap `json:"pokemon"`
}

// HoldableItem is one team-builder picker option. ShowdownID is what
// PokemonTeamMember.item stores and what the battle server accepts.
type HoldableItem struct {
	ShowdownID  string `json:"showdownId"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	ShortEffect string `json:"shortEffect"`
}
