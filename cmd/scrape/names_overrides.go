package main

// showdownSpeciesOverrides maps the PokeAPI slugs that do not normalise to
// Showdown's species id by lowercasing and stripping punctuation alone: the 46
// the UI and battle server already carried, plus the two Z-A Meowstic Megas,
// which neither table had (Showdown ids them "meowsticmmega" and
// "meowsticfmega", so those two team members used to fail as an unknown
// Pokemon).
// PokeAPI names some default formes explicitly where Showdown does not
// ("deoxys-normal" is Showdown's "deoxys", "zygarde-50" is "zygarde") and some
// formes differently ("meowstic-female" is "meowsticf", "necrozma-dusk" is
// "necrozmaduskmane", "maushold-family-of-three" is "mausholdfour").
//
// This is the one copy. It used to be hand-copied into the battle server
// (pokepedia-battle/src/speciesIds.ts) and the UI
// (pokepedia-ui/src/global/data/showdownSpecies.ts); both now read the
// showdown_id this scraper stores on every pokemon row instead.
var showdownSpeciesOverrides = map[string]string{
	"deoxys-normal":               "deoxys",
	"wormadam-plant":              "wormadam",
	"giratina-altered":            "giratina",
	"shaymin-land":                "shaymin",
	"basculin-red-striped":        "basculin",
	"darmanitan-standard":         "darmanitan",
	"darmanitan-galar-standard":   "darmanitangalar",
	"tornadus-incarnate":          "tornadus",
	"thundurus-incarnate":         "thundurus",
	"landorus-incarnate":          "landorus",
	"keldeo-ordinary":             "keldeo",
	"meloetta-aria":               "meloetta",
	"meowstic-male":               "meowstic",
	"meowstic-female":             "meowsticf",
	"aegislash-shield":            "aegislash",
	"pumpkaboo-average":           "pumpkaboo",
	"gourgeist-average":           "gourgeist",
	"zygarde-50":                  "zygarde",
	"oricorio-baile":              "oricorio",
	"lycanroc-midday":             "lycanroc",
	"wishiwashi-solo":             "wishiwashi",
	"mimikyu-disguised":           "mimikyu",
	"toxtricity-amped":            "toxtricity",
	"eiscue-ice":                  "eiscue",
	"indeedee-male":               "indeedee",
	"indeedee-female":             "indeedeef",
	"morpeko-full-belly":          "morpeko",
	"urshifu-single-strike":       "urshifu",
	"urshifu-single-strike-gmax":  "urshifugmax",
	"basculegion-male":            "basculegion",
	"basculegion-female":          "basculegionf",
	"enamorus-incarnate":          "enamorus",
	"minior-red":                  "minior",
	"necrozma-dusk":               "necrozmaduskmane",
	"necrozma-dawn":               "necrozmadawnwings",
	"oinkologne-female":           "oinkolognef",
	"maushold-family-of-three":    "mausholdfour",
	"tauros-paldea-combat-breed":  "taurospaldeacombat",
	"tauros-paldea-blaze-breed":   "taurospaldeablaze",
	"tauros-paldea-aqua-breed":    "taurospaldeaaqua",
	"squawkabilly-blue-plumage":   "squawkabillyblue",
	"squawkabilly-yellow-plumage": "squawkabillyyellow",
	"squawkabilly-white-plumage":  "squawkabillywhite",
	"ogerpon-wellspring-mask":     "ogerponwellspring",
	"ogerpon-hearthflame-mask":    "ogerponhearthflame",
	"ogerpon-cornerstone-mask":    "ogerponcornerstone",
	"meowstic-male-mega":          "meowsticmmega",
	"meowstic-female-mega":        "meowsticfmega",
}

// showdownMoveOverrides are PokeAPI move slugs Showdown spells differently.
var showdownMoveOverrides = map[string]string{
	"vice-grip": "visegrip",
}

// knownNameGaps are rows with no single Showdown counterpart, so they keep a
// NULL showdown_id on purpose: PokeAPI has one Embody Aspect ability where
// Showdown has four, one per Ogerpon mask.
var knownNameGaps = map[string]string{
	"ability:embody-aspect": "Showdown splits Embody Aspect into four abilities, one per Ogerpon mask",
}
