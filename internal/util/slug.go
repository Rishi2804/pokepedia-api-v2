package util

import (
	"regexp"
	"strings"
)

// slugContractions inverts prefixReplacements in name.go.
var slugContractions = map[string]string{
	"alolan": "alola", "galarian": "galar",
	"hisuian": "hisui", "paldean": "paldea",
	"50%": "50", "10%": "10",
}

var nonSlugChar = regexp.MustCompile(`[^a-z0-9-]+`)

// Slugify does not reorder words (unlike FormatName) — the pg_trgm fallback
// search matches order-insensitively, so it isn't needed here.
func Slugify(query string) string {
	words := strings.Fields(strings.ToLower(query))
	for i, w := range words {
		if repl, ok := slugContractions[w]; ok {
			words[i] = repl
		}
	}
	slug := strings.Join(words, "-")
	slug = strings.ReplaceAll(slug, "'", "")
	return nonSlugChar.ReplaceAllString(slug, "")
}
