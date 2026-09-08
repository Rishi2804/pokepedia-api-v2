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

// MoveInfo's last five fields are Legends: Arceus/Z-A-only extras -- always
// nil for every other game. Legends: Arceus populates SecondLevel (its
// Mastery level), PowerStrong/PowerAgile (Strong/Agile Style power -- Agile
// Style has no accuracy column of its own on Bulbapedia; it shares Accuracy
// above), and AccuracyStrong (Strong Style's own accuracy). Legends: Z-A
// populates SecondLevel (its "Plus" level) and Cooldown in place of PP,
// leaving the other three nil -- see 000010_legends_move_values.up.sql.
type MoveInfo struct {
	ID             int32  `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	MoveClass      string `json:"moveClass"`
	Power          *int32 `json:"power"`
	Accuracy       *int32 `json:"accuracy"`
	PP             *int32 `json:"pp"`
	LevelLearned   int32  `json:"levelLearned"`
	SecondLevel    *int32 `json:"secondLevel,omitempty"`
	PowerStrong    *int32 `json:"powerStrong,omitempty"`
	PowerAgile     *int32 `json:"powerAgile,omitempty"`
	AccuracyStrong *int32 `json:"accuracyStrong,omitempty"`
	Cooldown       *int32 `json:"cooldown,omitempty"`
}
