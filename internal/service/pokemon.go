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
	return s.buildDetail(ctx, p)
}

// GetPokemonDetailByName mirrors getPokemonByName. GetPokemonRow and
// GetPokemonByNameRow select identical columns in the same order, so
// they're structurally convertible via a plain type conversion.
func (s *PokemonService) GetPokemonDetailByName(ctx context.Context, name string) (*dto.PokemonDetail, error) {
	p, err := s.q.GetPokemonByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, store.GetPokemonRow(p))
}

func (s *PokemonService) buildDetail(ctx context.Context, p store.GetPokemonRow) (*dto.PokemonDetail, error) {
	dexNumbers, err := s.q.GetDexNumbers(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	dexEntries, err := s.q.GetDexEntries(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	evolution, err := s.q.GetEvolutionChain(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	moves, err := s.q.GetPokemonMoves(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	abilities, err := s.q.GetPokemonAbilities(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	result := &dto.PokemonDetail{
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
	}

	for _, a := range abilities {
		result.Abilities = append(result.Abilities, dto.AbilityInfo{
			ID:         a.AbilityID,
			Name:       util.FormatName(a.AbilityName, false),
			IsHidden:   a.Hidden,
			GenRemoved: a.GenRemoved,
		})
	}

	// Dex entries: sort by real game release order (Game.ORDER) BEFORE
	// converting to display form — GameIndex expects the raw lowercase
	// db value, not the uppercase display form.
	sortedEntries := append([]store.Dexentry(nil), dexEntries...)
	sort.SliceStable(sortedEntries, func(i, j int) bool {
		return pokeenum.GameIndex(sortedEntries[i].Game) < pokeenum.GameIndex(sortedEntries[j].Game)
	})
	for _, d := range sortedEntries {
		result.DexEntries = append(result.DexEntries, dto.DexEntry{
			Game:  pokeenum.ToDisplay(d.Game),
			Entry: d.Entry,
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

	return result, nil
}

// groupMoveset mirrors mapToMovesetDtoHelper: group by version group
// (sorted by VersionGroup.ORDER), then by learn method (sorted by
// LearnMethod.ORDER), then moves sorted by level learned within each group.
func groupMoveset(moves []store.GetPokemonMovesRow) []dto.VersionMoveset {
	byVersion := map[string][]store.GetPokemonMovesRow{}
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
		byMethod := map[string][]store.GetPokemonMovesRow{}
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
