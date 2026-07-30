package util

import (
	"sort"
	"strings"
	"unicode"
)

var specialWords = map[string]bool{
	"mega": true, "gmax": true, "primal": true, "ash": true,
	"50": true, "10": true, "complete": true,
	"alola": true, "galar": true, "hisui": true, "paldea": true,
}

var prefixReplacements = map[string]string{
	"alola": "alolan", "galar": "galarian", "hisui": "hisuian",
	"paldea": "paldean", "50": "50%", "10": "10%",
}

// FormatName: hardcoded exceptions for the
// jangmo-o line and Ho-Oh, "special word sorts first" for forms like
// Mega/Gmax/regional variants (e.g. "venusaur-mega" -> "Mega Venusaur"),
// then title-cases every word.
func FormatName(name string, isPokemon bool) string {
	words := strings.Split(name, "-")

	if isPokemon {
		switch name {
		case "jangmo-o", "hakamo-o", "kommo-o":
			return capitalizeFirst(name)
		case "ho-oh":
			return "Ho-Oh"
		}

		sort.SliceStable(words, func(i, j int) bool {
			iSpecial, jSpecial := specialWords[words[i]], specialWords[words[j]]
			if iSpecial == jSpecial {
				return false
			}
			return iSpecial
		})

		if repl, ok := prefixReplacements[words[0]]; ok {
			words[0] = repl
		}
	}

	for i, w := range words {
		words[i] = capitalizeFirst(w)
	}
	return strings.Join(words, " ")
}

func capitalizeFirst(word string) string {
	if word == "" {
		return word
	}
	r := []rune(word)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
