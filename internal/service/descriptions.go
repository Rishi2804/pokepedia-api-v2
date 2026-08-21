package service

import (
	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
)

// toDescriptions maps grouped description rows onto the wire DTO, converting
// each raw game slug ("lets-go-pikachu") to its display form
// ("LETS_GO_PIKACHU"). The rows already arrive grouped by text and ordered by
// release date — public.game is declared in release order, so the SQL
// array_agg and min() do that work.
//
// Move and ability descriptions share this one path; the only difference
// between them is which generated row type they start from, which the get
// callback resolves. Pokedex descriptions are not grouped server-side (the
// UI collapses runs of identical text per generation), so they're assembled
// directly in service/pokemon.go instead of through this helper.
func toDescriptions[T any](rows []T, get func(T) (games []string, text string)) []dto.Description {
	// Pre-initialized (not nil) so an empty result serializes as [] rather
	// than Go's default null, matching the rest of the DTOs.
	result := make([]dto.Description, 0, len(rows))
	for _, r := range rows {
		games, text := get(r)
		display := make([]string, len(games))
		for i, g := range games {
			display[i] = pokeenum.ToDisplay(g)
		}
		result = append(result, dto.Description{Games: display, Text: text})
	}
	return result
}
