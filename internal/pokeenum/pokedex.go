package pokeenum

import "fmt"

// PokedexRegionInfo mirrors one PokedexRegion enum constant.
type PokedexRegionInfo struct {
	String string // db value, e.g. "kalos-central"
	Name   string // display name, e.g. "Kalos Central"
	Gen    *int32 // nil for NATIONAL
}

var pokedexRegions = []PokedexRegionInfo{
	{"national", "National", nil},
	{"kanto", "Kanto", int32Ptr(1)},
	{"original-johto", "Johto", int32Ptr(2)},
	{"hoenn", "Hoenn", int32Ptr(3)},
	{"original-sinnoh", "Sinnoh", int32Ptr(4)},
	{"extended-sinnoh", "Sinnoh", int32Ptr(4)},
	{"updated-johto", "Johto", int32Ptr(4)},
	{"original-unova", "Unova", int32Ptr(5)},
	{"updated-unova", "Unova", int32Ptr(5)},
	{"kalos-central", "Kalos Central", int32Ptr(6)},
	{"kalos-coastal", "Kalos Coastal", int32Ptr(6)},
	{"kalos-mountain", "Kalos Mountain", int32Ptr(6)},
	{"updated-hoenn", "Hoenn", int32Ptr(6)},
	{"original-alola", "Alola", int32Ptr(7)},
	{"original-melemele", "Melemele", int32Ptr(7)},
	{"original-akala", "Akala", int32Ptr(7)},
	{"original-ulaula", "Ulaula", int32Ptr(7)},
	{"original-poni", "Poni", int32Ptr(7)},
	{"updated-alola", "Alola", int32Ptr(7)},
	{"updated-melemele", "Melemele", int32Ptr(7)},
	{"updated-akala", "Akala", int32Ptr(7)},
	{"updated-ulaula", "Ulaula", int32Ptr(7)},
	{"updated-poni", "Poni", int32Ptr(7)},
	{"letsgo-kanto", "Kanto", int32Ptr(7)},
	{"galar", "Galar", int32Ptr(8)},
	{"isle-of-armor", "Isle of Armor", int32Ptr(8)},
	{"crown-tundra", "Crown Tundra", int32Ptr(8)},
	{"hisui", "Hisui", int32Ptr(8)},
	{"paldea", "Paldea", int32Ptr(9)},
	{"kitakami", "Kitakami", int32Ptr(9)},
	{"blueberry", "Blueberry", int32Ptr(9)},
}

func int32Ptr(v int32) *int32 { return &v }

func GetPokedexRegion(dbValue string) (PokedexRegionInfo, error) {
	for _, r := range pokedexRegions {
		if r.String == dbValue {
			return r, nil
		}
	}
	return PokedexRegionInfo{}, fmt.Errorf("no PokedexRegion with name %s found", dbValue)
}

// PokedexVersionInfo mirrors one PokedexVersion enum constant.
type PokedexVersionInfo struct {
	String  string
	Regions []string // PokedexRegion db values
}

var pokedexVersions = []PokedexVersionInfo{
	{"national", []string{"national"}},
	{"red-blue-yellow", []string{"kanto"}},
	{"gold-silver-crystal", []string{"original-johto"}},
	{"ruby-sapphire-emerald", []string{"hoenn"}},
	{"firered-leafgreen", []string{"kanto"}},
	{"diamond-pearl-platinum", []string{"extended-sinnoh"}},
	{"heartgold-soulsilver", []string{"updated-johto"}},
	{"black-white", []string{"original-unova"}},
	{"black-2-white-2", []string{"updated-unova"}},
	{"x-y", []string{"kalos-central", "kalos-coastal", "kalos-mountain"}},
	{"omega-ruby-alpha-sapphire", []string{"updated-hoenn"}},
	{"sun-moon", []string{"original-alola", "original-melemele", "original-akala", "original-ulaula", "original-poni"}},
	{"ultra-sun-ultra-moon", []string{"updated-alola", "updated-melemele", "updated-akala", "updated-ulaula", "updated-poni"}},
	{"lets-go-pikachu-eevee", []string{"letsgo-kanto"}},
	{"sword-shield", []string{"galar", "isle-of-armor", "crown-tundra"}},
	{"brilliant-diamond-shining-pearl", []string{"original-sinnoh"}},
	{"legends-arceus", []string{"hisui"}},
	{"scarlet-violet", []string{"paldea", "kitakami", "blueberry"}},
}

func GetPokedexVersion(dbValue string) (PokedexVersionInfo, error) {
	for _, v := range pokedexVersions {
		if v.String == dbValue {
			return v, nil
		}
	}
	return PokedexVersionInfo{}, fmt.Errorf("no PokedexVersion with name %s found", dbValue)
}

// VersionGroupInfo mirrors one VersionGroup enum constant.
type VersionGroupInfo struct {
	VersionName string
	Gen         int32
	Regions     []string // PokedexRegion db values
}

var versionGroups = []VersionGroupInfo{
	{"national", 10, []string{"national"}},
	{"red-blue", 1, []string{"kanto"}},
	{"yellow", 1, []string{"kanto"}},
	{"gold-silver", 2, []string{"original-johto"}},
	{"crystal", 2, []string{"original-johto"}},
	{"ruby-sapphire", 3, []string{"hoenn"}},
	{"emerald", 3, []string{"hoenn"}},
	{"firered-leafgreen", 3, []string{"kanto"}},
	{"diamond-pearl", 4, []string{"original-sinnoh"}},
	{"platinum", 4, []string{"extended-sinnoh"}},
	{"heartgold-soulsilver", 4, []string{"updated-johto"}},
	{"black-white", 5, []string{"original-unova"}},
	{"black-2-white-2", 5, []string{"updated-unova"}},
	{"x-y", 6, []string{"kalos-central", "kalos-coastal", "kalos-mountain"}},
	{"omega-ruby-alpha-sapphire", 6, []string{"updated-hoenn"}},
	{"sun-moon", 7, []string{"original-alola", "original-melemele", "original-akala", "original-ulaula", "original-poni"}},
	{"ultra-sun-ultra-moon", 7, []string{"updated-alola", "updated-melemele", "updated-akala", "updated-ulaula", "updated-poni"}},
	{"lets-go-pikachu-lets-go-eevee", 7, []string{"letsgo-kanto"}},
	{"sword-shield", 8, []string{"galar", "isle-of-armor", "crown-tundra"}},
	{"brilliant-diamond-and-shining-pearl", 8, []string{"original-sinnoh"}},
	{"legends-arceus", 8, []string{"hisui"}},
	{"scarlet-violet", 9, []string{"paldea", "kitakami", "blueberry"}},
}

func GetVersionGroup(dbValue string) (VersionGroupInfo, error) {
	for _, v := range versionGroups {
		if v.VersionName == dbValue {
			return v, nil
		}
	}
	return VersionGroupInfo{}, fmt.Errorf("no VersionGroup with name %s found", dbValue)
}
