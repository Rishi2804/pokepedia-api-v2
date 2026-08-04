package service

import (
	"context"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

type AbilitiesService struct {
	q *store.Queries
}

func NewAbilitiesService(q *store.Queries) *AbilitiesService {
	return &AbilitiesService{q: q}
}

func (s *AbilitiesService) GetAbilityList(ctx context.Context) ([]dto.AbilitySnap, error) {
	rows, err := s.q.GetAbilitiesList(ctx)
	if err != nil {
		return nil, err
	}
	response := []dto.AbilitySnap{}
	for _, r := range rows {
		response = append(response, dto.AbilitySnap{
			ID:   r.ID,
			Name: r.Name,
			Gen:  r.Gen,
		})
	}
	return response, nil
}

func (s *AbilitiesService) GetAbilityDetail(ctx context.Context, id int32) (*dto.AbilityDetail, error) {
	a, err := s.q.GetAbility(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, a)
}

func (s *AbilitiesService) GetAbilityByName(ctx context.Context, name string) (*dto.AbilityDetail, error) {
	a, err := s.q.GetAbilityByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, store.GetAbilityRow(a))
}

func (s *AbilitiesService) buildDetail(ctx context.Context, a store.GetAbilityRow) (*dto.AbilityDetail, error) {
	descriptions, err := parseDescriptions(a.Descriptions)
	if err != nil {
		return nil, err
	}

	pokemon, err := s.q.GetPokemonLearnableAbility(ctx, a.ID)
	if err != nil {
		return nil, err
	}

	pokemonResponse := []dto.PokemonSnap{}
	for _, p := range pokemon {
		pokemonResponse = append(pokemonResponse, dto.PokemonSnap{
			SpeciesID: p.SpeciesID,
			ID:        p.ID,
			Name:      util.FormatName(p.Name, true),
			Type1:     pokeenum.ToDisplay(p.Type1),
			Type2:     nilIfEmptyPtr(p.Type2),
		})
	}

	response := &dto.AbilityDetail{
		ID:           a.ID,
		Name:         util.FormatName(a.Name, false),
		Gen:          a.Gen,
		Descriptions: descriptions,
		Pokemon:      pokemonResponse,
	}

	if a.Effect != nil {
		response.Effect = *a.Effect
	}

	return response, nil
}
