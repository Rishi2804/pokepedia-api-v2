package service

import (
	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
)

// toDescriptions converts each row's game slugs to display form via
// pokeenum.ToDisplay. Shared by move/ability descriptions; Pokedex
// descriptions aren't grouped server-side, so they're assembled directly in
// service/pokemon.go instead.
func toDescriptions[T any](rows []T, get func(T) (games []string, text string)) []dto.Description {
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
