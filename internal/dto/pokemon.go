package dto

type PokemonDetail struct {
	ID           int32                `json:"id"`
	SpeciesID    int32                `json:"speciesId"`
	Name         string               `json:"name"`
	Gen          int32                `json:"gen"`
	Type1        string               `json:"type1"`
	Type2        *string              `json:"type2"`
	Abilities    []AbilityInfo        `json:"abilities"`
	Weight       float64              `json:"weight"`
	Height       float64              `json:"height"`
	GenderRate   int32                `json:"genderRate"`
	Stats        Stats                `json:"stats"`
	Forms        []string             `json:"forms"`
	Descriptions []PokemonDescription `json:"descriptions"`
	DexNumbers   []DexNumberInfo      `json:"dexNumbers"`
	Evolution    []EvolutionLine      `json:"evolutionChain"`
	Movesets     []VersionMoveset     `json:"movesets"`
}

// PokemonDescription is one Pokedex entry for one game. Unlike move and ability
// descriptions, these are not grouped server-side — the UI collapses runs of
// identical text per generation, so it needs them per game.
type PokemonDescription struct {
	Game string `json:"game"`
	Text string `json:"text"`
}

type Stats struct {
	HP    int32 `json:"hp"`
	Atk   int32 `json:"atk"`
	Def   int32 `json:"def"`
	SpAtk int32 `json:"spatk"`
	SpDef int32 `json:"spdef"`
	Speed int32 `json:"speed"`
	BST   int32 `json:"bst"`
}

type AbilityInfo struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	IsHidden   bool   `json:"isHidden"`
	GenRemoved *int32 `json:"genRemoved"`
}

type DexNumberInfo struct {
	DexName   string `json:"dexName"`
	DexNumber int32  `json:"dexNumber"`
}

type EvolutionLine struct {
	ID          int32    `json:"id"`
	FromPokemon int32    `json:"fromPokemon"`
	FromDisplay string   `json:"fromDisplay"`
	ToPokemon   int32    `json:"toPokemon"`
	ToDisplay   string   `json:"toDisplay"`
	Details     []string `json:"details"`
	Region      *string  `json:"region"`
	AltForm     int32    `json:"altForm"`
}

type VersionMoveset struct {
	VersionGroup string           `json:"versionGroup"`
	Methods      []LearnMethodSet `json:"learnMethodSets"`
}

type LearnMethodSet struct {
	Method string     `json:"method"`
	Moves  []MoveInfo `json:"moves"`
}

type MoveInfo struct {
	ID           int32  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	MoveClass    string `json:"moveClass"`
	Power        *int32 `json:"power"`
	Accuracy     *int32 `json:"accuracy"`
	PP           *int32 `json:"pp"`
	LevelLearned int32  `json:"levelLearned"`
}
