package service

import (
	"context"
	"sort"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

type PokemonService struct {
	q *store.Queries
}

func NewPokemonService(q *store.Queries) *PokemonService {
	return &PokemonService{q: q}
}

// GetPokemonDetail mirrors PokemonServiceImpl.getPokemonById: fetch the
// base row, then compose five related datasets into one response.
func (s *PokemonService) GetPokemonDetail(ctx context.Context, id int32) (*dto.PokemonDetail, error) {
	p, err := s.q.GetPokemon(ctx, id)
	if err != nil {
		return nil, err
	}
	details, err := s.buildDetails(ctx, []store.GetPokemonRow{p})
	if err != nil {
		return nil, err
	}
	return &details[0], nil
}

// GetPokemonDetailByName mirrors getPokemonByName. GetPokemonRow and
// GetPokemonByNameRow select identical columns in the same order, so
// they're structurally convertible via a plain type conversion.
func (s *PokemonService) GetPokemonDetailByName(ctx context.Context, name string) (*dto.PokemonDetail, error) {
	p, err := s.q.GetPokemonByName(ctx, name)
	if err != nil {
		return nil, err
	}
	details, err := s.buildDetails(ctx, []store.GetPokemonRow{store.GetPokemonRow(p)})
	if err != nil {
		return nil, err
	}
	return &details[0], nil
}

// buildDetails composes the detail response for a batch of pokemon rows using
// five bulk queries (one round trip each) instead of five per pokemon.
func (s *PokemonService) buildDetails(ctx context.Context, ps []store.GetPokemonRow) ([]dto.PokemonDetail, error) {
	ids := make([]int32, len(ps))
	for i, p := range ps {
		ids[i] = p.ID
	}

	dexNumbers, err := s.q.GetDexNumbersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	dexEntries, err := s.q.GetPokemonDescriptionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	evolution, err := s.q.GetEvolutionChainByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	moves, err := s.q.GetPokemonMovesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	abilities, err := s.q.GetPokemonAbilitiesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	dexNumbersByID := map[int32][]store.GetDexNumbersByIDsRow{}
	for _, d := range dexNumbers {
		if d.PokemonID == nil {
			continue
		}
		dexNumbersByID[*d.PokemonID] = append(dexNumbersByID[*d.PokemonID], d)
	}
	dexEntriesByID := map[int32][]store.Pokemondescription{}
	for _, d := range dexEntries {
		dexEntriesByID[d.PokemonID] = append(dexEntriesByID[d.PokemonID], d)
	}
	evolutionByID := map[int32][]store.GetEvolutionChainByIDsRow{}
	for _, e := range evolution {
		if e.PokemonID == nil {
			continue
		}
		evolutionByID[*e.PokemonID] = append(evolutionByID[*e.PokemonID], e)
	}
	movesByID := map[int32][]store.GetPokemonMovesByIDsRow{}
	for _, m := range moves {
		movesByID[m.PokemonID] = append(movesByID[m.PokemonID], m)
	}
	abilitiesByID := map[int32][]store.GetPokemonAbilitiesByIDsRow{}
	for _, a := range abilities {
		abilitiesByID[a.PokemonID] = append(abilitiesByID[a.PokemonID], a)
	}

	results := make([]dto.PokemonDetail, len(ps))
	for i, p := range ps {
		results[i] = assembleDetail(p, dexNumbersByID[p.ID], dexEntriesByID[p.ID], evolutionByID[p.ID], movesByID[p.ID], abilitiesByID[p.ID])
	}
	return results, nil
}

// assembleDetail is the pure assembly step split out of the old buildDetail:
// no ctx, no DB, just DTO composition from already-fetched rows.
func assembleDetail(p store.GetPokemonRow, dexNumbers []store.GetDexNumbersByIDsRow, dexEntries []store.Pokemondescription, evolution []store.GetEvolutionChainByIDsRow, moves []store.GetPokemonMovesByIDsRow, abilities []store.GetPokemonAbilitiesByIDsRow) dto.PokemonDetail {
	result := dto.PokemonDetail{
		ID:         p.ID,
		SpeciesID:  p.SpeciesID,
		Name:       util.FormatName(p.Name, true),
		Gen:        p.Gen,
		Type1:      pokeenum.ToDisplay(p.Type1),
		Type2:      displayPtr(p.Type2),
		GenderRate: p.GenderRate,
		Weight:     p.Weight,
		Height:     p.Height,
		Forms:      p.Forms,
		Stats: dto.Stats{
			HP: p.Hp, Atk: p.Atk, Def: p.Def,
			SpAtk: p.Spatk, SpDef: p.Spdef, Speed: p.Speed, BST: p.Bst,
		},
		// Pre-initialized (not nil) so an empty result serializes as [] to
		// match legacy behavior, instead of Go's default null for nil slices.
		Abilities:    []dto.AbilityInfo{},
		Descriptions: []dto.PokemonDescription{},
		DexNumbers:   []dto.DexNumberInfo{},
		Evolution:    []dto.EvolutionLine{},
	}

	for _, a := range abilities {
		result.Abilities = append(result.Abilities, dto.AbilityInfo{
			ID:         a.AbilityID,
			Name:       util.FormatName(a.AbilityName, false),
			IsHidden:   a.Hidden,
			GenRemoved: a.GenRemoved,
		})
	}

	// Ordered by release date in SQL — public.game is declared in release
	// order, so no Go-side sort is needed. One entry per game; the UI collapses
	// runs of identical text within a generation.
	for _, d := range dexEntries {
		result.Descriptions = append(result.Descriptions, dto.PokemonDescription{
			Game: pokeenum.ToDisplay(d.Version),
			Text: d.Text,
		})
	}

	for _, d := range dexNumbers {
		result.DexNumbers = append(result.DexNumbers, dto.DexNumberInfo{
			DexName:   pokeenum.ToDisplay(d.Name),
			DexNumber: d.Num,
		})
	}

	for _, e := range evolution {
		result.Evolution = append(result.Evolution, dto.EvolutionLine{
			ID:          e.ID,
			FromPokemon: e.FromPokemon,
			FromDisplay: util.FormatName(e.FromDisplay, true),
			ToPokemon:   e.ToPokemon,
			ToDisplay:   util.FormatName(e.ToDisplay, true),
			Details:     e.Details,
			Region:      e.Region,
			AltForm:     e.AltForm,
		})
	}

	result.Movesets = groupMoveset(moves)

	return result
}

// groupMoveset mirrors mapToMovesetDtoHelper: group by version group
// (sorted by VersionGroup.ORDER), then by learn method (sorted by
// LearnMethod.ORDER), then moves sorted by level learned within each group.
func groupMoveset(moves []store.GetPokemonMovesByIDsRow) []dto.VersionMoveset {
	byVersion := map[string][]store.GetPokemonMovesByIDsRow{}
	for _, m := range moves {
		if m.Version == nil {
			continue // schema allows null; shouldn't occur in practice
		}
		byVersion[*m.Version] = append(byVersion[*m.Version], m)
	}

	versions := []string{}
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		return pokeenum.VersionGroupIndex(versions[i]) < pokeenum.VersionGroupIndex(versions[j])
	})

	result := []dto.VersionMoveset{}
	for _, v := range versions {
		byMethod := map[string][]store.GetPokemonMovesByIDsRow{}
		for _, m := range byVersion[v] {
			byMethod[m.Method] = append(byMethod[m.Method], m)
		}

		methods := []string{}
		for m := range byMethod {
			methods = append(methods, m)
		}
		sort.Slice(methods, func(i, j int) bool {
			return pokeenum.LearnMethodIndex(methods[i]) < pokeenum.LearnMethodIndex(methods[j])
		})

		methodSets := []dto.LearnMethodSet{}
		for _, method := range methods {
			moveList := byMethod[method]
			sort.SliceStable(moveList, func(i, j int) bool {
				return moveList[i].LevelLearned < moveList[j].LevelLearned
			})

			moveInfos := []dto.MoveInfo{}
			for _, m := range moveList {
				moveInfos = append(moveInfos, dto.MoveInfo{
					ID:           m.MoveID,
					Name:         util.FormatName(m.Name, false),
					Type:         pokeenum.ToDisplay(m.Type),
					MoveClass:    pokeenum.ToDisplay(m.Class),
					Power:        m.Power,
					Accuracy:     m.Accuracy,
					PP:           m.Pp,
					LevelLearned: m.LevelLearned,
				})
			}
			methodSets = append(methodSets, dto.LearnMethodSet{
				Method: pokeenum.ToDisplay(method),
				Moves:  moveInfos,
			})
		}
		result = append(result, dto.VersionMoveset{
			VersionGroup: pokeenum.ToDisplay(v),
			Methods:      methodSets,
		})
	}
	return result
}
