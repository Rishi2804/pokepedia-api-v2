package service

import (
	"context"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
)

type SpeciesService struct {
	q              *store.Queries
	pokemonService *PokemonService
}

func NewSpeciesService(q *store.Queries, pokemonService *PokemonService) *SpeciesService {
	return &SpeciesService{q: q, pokemonService: pokemonService}
}

// GetSpeciesDetail mirrors getPokemonFromSpeciesId: IDs above 10000 are
// Pokemon (variant) IDs, not species IDs, and need resolving to their
// real species_id first.
func (s *SpeciesService) GetSpeciesDetail(ctx context.Context, id int32) (*dto.SpeciesDetail, error) {
	finalID := id
	if id > 10000 {
		speciesID, err := s.q.GetSpeciesIdFromPokemon(ctx, id)
		if err != nil {
			return nil, err
		}
		finalID = speciesID
	}

	sp, err := s.q.GetSpeciesByID(ctx, finalID)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, sp.ID, sp.Name)
}

func (s *SpeciesService) GetSpeciesDetailByName(ctx context.Context, name string) (*dto.SpeciesDetail, error) {
	sp, err := s.q.GetSpeciesByName(ctx, &name)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, sp.ID, sp.Name)
}

func (s *SpeciesService) buildDetail(ctx context.Context, id int32, name *string) (*dto.SpeciesDetail, error) {
	rows, err := s.q.GetPokemonBySpeciesID(ctx, id)
	if err != nil {
		return nil, err
	}

	displayName := ""
	if name != nil {
		displayName = *name
	}

	result := &dto.SpeciesDetail{ID: id, Name: displayName, Pokemon: []dto.PokemonDetail{}}

	ps := make([]store.GetPokemonRow, len(rows))
	for i, r := range rows {
		ps[i] = store.GetPokemonRow(r)
	}
	details, err := s.pokemonService.buildDetails(ctx, ps)
	if err != nil {
		return nil, err
	}
	result.Pokemon = append(result.Pokemon, details...)

	return result, nil
}
