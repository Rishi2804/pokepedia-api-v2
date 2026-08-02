package service

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

type MovesService struct {
	q *store.Queries
}

func NewMovesService(q *store.Queries) *MovesService {
	return &MovesService{q: q}
}

func (s *MovesService) GetMoveList(ctx context.Context) ([]dto.MoveSnap, error) {
	rows, err := s.q.GetMoves(ctx)
	if err != nil {
		return nil, err
	}
	response := []dto.MoveSnap{}
	for _, r := range rows {
		response = append(response, dto.MoveSnap{
			ID:       r.ID,
			Name:     util.FormatName(r.Name, false),
			Type:     pokeenum.ToDisplay(r.Type),
			Class:    pokeenum.ToDisplay(r.Class),
			Power:    r.Power,
			Accuracy: r.Accuracy,
			PP:       r.Pp,
			Gen:      r.Gen,
		})
	}
	return response, nil
}

func (s *MovesService) GetMoveDetail(ctx context.Context, id int32) (*dto.MoveDetail, error) {
	m, err := s.q.GetMove(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, m)
}

func (s *MovesService) GetMoveByName(ctx context.Context, name string) (*dto.MoveDetail, error) {
	m, err := s.q.GetMoveByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, store.GetMoveRow(m))
}

func (s *MovesService) buildDetail(ctx context.Context, m store.GetMoveRow) (*dto.MoveDetail, error) {
	pastVals, err := s.q.GetPastMoveValues(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	descriptions, err := parseDescriptions(m.Descriptions)
	if err != nil {
		return nil, err
	}
	pokemon, err := s.q.GetPokemonLearnableMove(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	response := &dto.MoveDetail{
		ID:             m.ID,
		Name:           util.FormatName(m.Name, false),
		Type:           pokeenum.ToDisplay(m.Type),
		Gen:            m.Gen,
		Class:          pokeenum.ToDisplay(m.Class),
		MovePower:      m.Power,
		MoveAccuracy:   m.Accuracy,
		MovePP:         m.Pp,
		Descriptions:   descriptions,
		PastMoveValues: []dto.PastMoveValues{},
	}

	if m.Effect != nil {
		response.Effect = *m.Effect
	}

	for _, r := range pastVals {
		versionGroups := make([]string, len(r.VersionGroups))
		for i, vg := range r.VersionGroups {
			versionGroups[i] = pokeenum.ToDisplay(vg)
		}
		response.PastMoveValues = append(response.PastMoveValues, dto.PastMoveValues{
			MovePower:    r.Power,
			MoveAccuracy: r.Accuracy,
			MovePP:       r.Pp,
			VersionGroup: versionGroups,
		})
	}

	response.Pokemon = groupPokemonLearnable(pokemon)

	return response, nil
}

func parseDescriptions(raw []string) ([]dto.Description, error) {
	result := []dto.Description{}
	for _, entryJSON := range raw {
		var parsed struct {
			Entry        string   `json:"entry"`
			VersionGroup []string `json:"versionGroup"`
		}
		if err := json.Unmarshal([]byte(entryJSON), &parsed); err != nil {
			return nil, err
		}
		displayGroups := make([]string, len(parsed.VersionGroup))
		for i, vg := range parsed.VersionGroup {
			displayGroups[i] = pokeenum.ToDisplay(vg)
		}
		result = append(result, dto.Description{
			Description:   parsed.Entry,
			VersionGroups: displayGroups,
		})
	}
	return result, nil
}

func groupPokemonLearnable(rows []store.GetPokemonLearnableMoveRow) []dto.PokemonLearnable {
	byMethod := map[string][]store.GetPokemonLearnableMoveRow{}
	for _, r := range rows {
		byMethod[r.Method] = append(byMethod[r.Method], r)
	}

	var methods []string
	for m := range byMethod {
		methods = append(methods, m)
	}
	sort.Slice(methods, func(i, j int) bool {
		return pokeenum.LearnMethodIndex(methods[i]) < pokeenum.LearnMethodIndex(methods[j])
	})

	result := []dto.PokemonLearnable{}
	for _, method := range methods {
		var pokemon []dto.PokemonSnap
		for _, r := range byMethod[method] {
			entry := dto.PokemonSnap{
				SpeciesID: r.SpeciesID,
				ID:        r.ID,
				Name:      util.FormatName(r.Name, true),
				Type1:     pokeenum.ToDisplay(r.Type1),
				Type2:     nilIfEmptyPtr(r.Type2),
			}
			pokemon = append(pokemon, entry)
		}
		result = append(result, dto.PokemonLearnable{
			Method:  pokeenum.ToDisplay(method),
			Pokemon: pokemon,
		})
	}
	return result
}
