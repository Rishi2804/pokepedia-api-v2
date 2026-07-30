package service

import (
	"context"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

type SpeciesService struct {
	q *store.Queries
}

func NewSpeciesService(q *store.Queries) *SpeciesService {
	return &SpeciesService{q: q}
}

func (s *SpeciesService) GetSpeciesDetail(ctx context.Context, id int32) (*dto.SpeciesDetail, error) {
	sp, err := s.q.GetSpeciesByID(ctx, id)
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
	rows, err := s.q.GetPokemonSummariesBySpeciesID(ctx, id)
	if err != nil {
		return nil, err
	}

	displayName := ""
	if name != nil {
		displayName = util.FormatName(*name, true)
	}

	result := &dto.SpeciesDetail{ID: id, Name: displayName}
	for _, p := range rows {
		summary := dto.PokemonSummary{
			ID:    p.ID,
			Name:  util.FormatName(p.Name, true),
			Gen:   p.Gen,
			Type1: pokeenum.ToDisplay(p.Type1),
		}
		if p.Type2 != nil && *p.Type2 != "" {
			t2 := pokeenum.ToDisplay(*p.Type2)
			summary.Type2 = &t2
		}
		result.Pokemon = append(result.Pokemon, summary)
	}
	return result, nil
}
