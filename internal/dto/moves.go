package dto

// Slug is the URL form (/move/u-turn) and ShowdownID Showdown's own id; Name
// is the display name, which can't be turned back into either ("King's
// Shield" is "kings-shield").
type MoveSnap struct {
	ID         int32  `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	ShowdownID string `json:"showdownId"`
	Type       string `json:"type"`
	Class      string `json:"moveClass"`
	Power      *int32 `json:"power"`
	Accuracy   *int32 `json:"accuracy"`
	PP         *int32 `json:"pp"`
	Gen        int32  `json:"gen"`
}

type MoveDetail struct {
	ID             int32              `json:"id"`
	Slug           string             `json:"slug"`
	Name           string             `json:"name"`
	ShowdownID     string             `json:"showdownId"`
	Type           string             `json:"type"`
	Gen            int32              `json:"gen"`
	Class          string             `json:"moveClass"`
	MovePower      *int32             `json:"movePower"`
	MoveAccuracy   *int32             `json:"moveAccuracy"`
	MovePP         *int32             `json:"movePP"`
	PastMoveValues []PastMoveValues   `json:"pastMoveValues"`
	Effect         string             `json:"effect"`
	Descriptions   []Description      `json:"descriptions"`
	Pokemon        []PokemonLearnable `json:"pokemon"`
}

type PastMoveValues struct {
	MovePower    *int32   `json:"movePower"`
	MoveAccuracy *int32   `json:"moveAccuracy"`
	MovePP       *int32   `json:"movePP"`
	VersionGroup []string `json:"versionGroups"`
}

// Description is the one shape every versioned text uses — Pokedex flavor,
// move descriptions and ability descriptions alike. Games is the set of game
// versions sharing this exact wording, in release order.
type Description struct {
	Games []string `json:"games"`
	Text  string   `json:"text"`
}

type PokemonLearnable struct {
	Method  string        `json:"method"`
	Pokemon []PokemonSnap `json:"pokemon"`
}

type PokemonSnap struct {
	SpeciesID int32   `json:"speciesId"`
	ID        int32   `json:"id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Type1     string  `json:"type1"`
	Type2     *string `json:"type2"`
}
