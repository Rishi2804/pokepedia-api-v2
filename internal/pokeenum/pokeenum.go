package pokeenum

import "strings"

// ToDisplay converts a lowercase-hyphenated DB value (e.g. "level-up",
// "black-2-white-2") into the same uppercase-underscore form Java's
// Enum.name() produces (e.g. "LEVEL_UP", "BLACK_2_WHITE_2"). Type,
// MoveClass, LearnMethod, Game, and VersionGroup all follow this transform.
func ToDisplay(dbValue string) string {
	return strings.ToUpper(strings.ReplaceAll(dbValue, "-", "_"))
}

// LearnMethodOrder mirrors LearnMethod's declared order (LearnMethod.ORDER).
var LearnMethodOrder = []string{
	"level-up", "machine", "tutor", "egg", "light-ball-egg", "form-change", "zygarde-cube",
}

// Game release order now lives in the public.game enum's declaration order, so
// the database sorts description rows and no Go-side GameOrder/GameIndex pair
// is needed.

// VersionGroupOrder mirrors VersionGroup's declared order (VersionGroup.ORDER).
var VersionGroupOrder = []string{
	"national", "red-blue", "yellow", "gold-silver", "crystal", "ruby-sapphire",
	"emerald", "firered-leafgreen", "diamond-pearl", "platinum",
	"heartgold-soulsilver", "black-white", "black-2-white-2", "x-y",
	"omega-ruby-alpha-sapphire", "sun-moon", "ultra-sun-ultra-moon",
	"lets-go-pikachu-lets-go-eevee", "sword-shield",
	"brilliant-diamond-and-shining-pearl", "legends-arceus", "scarlet-violet",
}

func indexOf(list []string, value string) int {
	for i, v := range list {
		if v == value {
			return i
		}
	}
	return len(list) // unrecognized values sort last instead of panicking
}

func LearnMethodIndex(dbValue string) int  { return indexOf(LearnMethodOrder, dbValue) }
func VersionGroupIndex(dbValue string) int { return indexOf(VersionGroupOrder, dbValue) }
