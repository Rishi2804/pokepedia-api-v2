package service

import (
	"context"
	"fmt"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

type PokedexService struct {
	q *store.Queries
}

func NewPokedexService(q *store.Queries) *PokedexService {
	return &PokedexService{q: q}
}

func formatNullableName(name *string) string {
	if name == nil {
		return ""
	}
	return util.FormatName(*name, true)
}

func (s *PokedexService) GetDexByVersion(ctx context.Context, versionName string) ([]dto.PokedexRegionGroup, error) {
	pv, err := pokeenum.GetPokedexVersion(versionName)
	if err != nil {
		return nil, err
	}

	response := []dto.PokedexRegionGroup{}

	for _, regionName := range pv.Regions {
		if versionName == "national" {
			rows, err := s.q.GetDexNational(ctx)
			if err != nil {
				return nil, err
			}

			var fullDex []dto.PokedexEntry
			for _, r := range rows {
				fullDex = append(fullDex, dto.PokedexEntry{
					DexNumber: r.SpeciesID, SpeciesID: r.SpeciesID, PokemonID: r.ID,
					Name: formatNullableName(r.Name), Gen: r.Gen,
					Type1: pokeenum.ToDisplay(r.Type1),
					Type2: nilIfEmptyPtr(r.Type2),
				})
			}

			// Official national dex numbers where each generation begins.
			separatorNums := []int32{1, 152, 252, 387, 494, 650, 722, 810, 906}
			separatorIndices := make([]int, len(separatorNums))
			for i, num := range separatorNums {
				separatorIndices[i] = -1
				for j, entry := range fullDex {
					if entry.DexNumber == num {
						separatorIndices[i] = j
						break
					}
				}
				if separatorIndices[i] == -1 {
					return nil, fmt.Errorf("could not find National Dex #%d", num)
				}
			}

			for i := range separatorIndices {
				start := separatorIndices[i]
				end := len(fullDex)
				if i+1 < len(separatorIndices) {
					end = separatorIndices[i+1]
				}

				response = append(response, dto.PokedexRegionGroup{
					Name:    fmt.Sprintf("Gen %d", i+1),
					Pokemon: fullDex[start:end],
				})
			}

			return response, nil
		}

		region, err := pokeenum.GetPokedexRegion(regionName)
		if err != nil {
			return nil, err
		}
		rows, err := s.q.GetDexByRegion(ctx, store.GetDexByRegionParams{
			Gen:  *region.Gen,
			Name: store.Dex(regionName),
		})
		if err != nil {
			return nil, err
		}
		var entries []dto.PokedexEntry
		for _, r := range rows {
			entries = append(entries, dto.PokedexEntry{
				DexNumber: r.Num, SpeciesID: r.SpeciesID, PokemonID: r.ID,
				Name: formatNullableName(r.Name), Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1),
				Type2: nilIfEmptyPtr(r.Type2),
			})
		}
		response = append(response, dto.PokedexRegionGroup{
			Name: region.Name, Pokemon: entries,
		})
	}

	return response, nil
}

// GetTeamCandidates mirrors getTeamCandidates/getTeamCandidatesNational:
// resolve to a VersionGroup, then combine candidates across every region
// it spans, deduplicated by Pokemon ID (a Pokemon can legitimately appear
// in more than one sub-region of a multi-region version group).
func (s *PokedexService) GetTeamCandidates(ctx context.Context, versionName string) ([]dto.TeamBuildingGroup, error) {
	vg, err := pokeenum.GetVersionGroup(versionName)
	if err != nil {
		return nil, err
	}

	if versionName == "national" {
		rows, err := s.q.GetTeamCandidatesNational(ctx)
		if err != nil {
			return nil, err
		}

		candidates := []dto.TeamCandidate{}
		for _, r := range rows {
			plain := dto.TeamCandidate{
				ID: r.ID, Name: util.FormatName(*r.Name, true), Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1), Type2: nilIfEmptyPtr(r.Type2), GenderRate: r.GenderRate,
			}
			detail, err := s.buildCandidateDetail(ctx, plain, versionName, vg.Gen)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, detail)
		}

		return []dto.TeamBuildingGroup{
			{ListName: "National", Pokemon: candidates},
		}, nil
	}

	seen := map[int32]bool{}
	var result []dto.TeamBuildingGroup

	for _, regionName := range vg.Regions {
		region, err := pokeenum.GetPokedexRegion(regionName)
		if err != nil {
			return nil, err
		}
		rows, err := s.q.GetTeamCandidatesByRegion(ctx, store.GetTeamCandidatesByRegionParams{
			Gen:  *region.Gen,
			Name: store.Dex(regionName),
		})
		if err != nil {
			return nil, err
		}

		candidates := []dto.TeamCandidate{}
		for _, r := range rows {
			seen[r.ID] = true
			plain := dto.TeamCandidate{
				ID: r.ID, Name: util.FormatName(r.Name, true), Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1), Type2: nilIfEmptyPtr(r.Type2), GenderRate: r.GenderRate,
			}
			detail, err := s.buildCandidateDetail(ctx, plain, versionName, vg.Gen)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, detail)
		}

		result = append(result, dto.TeamBuildingGroup{
			ListName: region.Name, Pokemon: candidates,
		})
	}

	nationalRows, err := s.q.GetTeamCandidatesNationalByVersionGroup(ctx, store.GetTeamCandidatesNationalByVersionGroupParams{
		Gen:     vg.Gen,
		Version: store.Group(versionName),
	})
	if err != nil {
		return nil, err
	}

	national := []dto.TeamCandidate{}
	for _, r := range nationalRows {
		if seen[r.ID] {
			continue
		}
		plain := dto.TeamCandidate{
			ID: r.ID, Name: util.FormatName(r.Name, true), Gen: r.Gen,
			Type1: pokeenum.ToDisplay(r.Type1), Type2: nilIfEmptyPtr(r.Type2), GenderRate: r.GenderRate,
		}
		detail, err := s.buildCandidateDetail(ctx, plain, versionName, vg.Gen)
		if err != nil {
			return nil, err
		}
		national = append(national, detail)
	}

	if len(national) > 0 {
		result = append(result, dto.TeamBuildingGroup{
			ListName: "National", Pokemon: national,
		})
	}

	return result, nil
}

func (s *PokedexService) buildCandidateDetail(ctx context.Context, cand dto.TeamCandidate, versionGroupName string, gen int32) (dto.TeamCandidate, error) {
	abilities, err := s.q.GetPokemonAbilities(ctx, cand.ID)
	if err != nil {
		return dto.TeamCandidate{}, err
	}
	moveRows, err := s.q.GetPokemonMoves(ctx, cand.ID)
	if err != nil {
		return dto.TeamCandidate{}, err
	}

	abilitiesDto := []dto.CandidateAbility{}

	var matched *store.GetPokemonAbilitiesRow
	for i := range abilities {
		if genRemovedValue(abilities[i].GenRemoved) != 0 {
			matched = &abilities[i]
			break
		}
	}

	if matched != nil && gen <= *matched.GenRemoved {
		hiddenRemoved := false
		for _, a := range abilities {
			if genRemovedValue(a.GenRemoved) != 0 && a.Hidden {
				hiddenRemoved = true
				break
			}
		}
		if hiddenRemoved {
			for _, a := range abilities {
				if genRemovedValue(a.GenRemoved) == 0 && a.Hidden {
					continue
				}
				abilitiesDto = append(abilitiesDto, dto.CandidateAbility{
					ID: a.AbilityID, Name: util.FormatName(a.AbilityName, false),
				})
			}
		} else {
			// special cases, ported verbatim
			switch cand.ID {
			case 94:
				abilitiesDto = append(abilitiesDto, dto.CandidateAbility{ID: 26, Name: "Levitate"})
			case 275:
				for _, a := range abilities {
					if a.AbilityID == 274 {
						continue
					}
					abilitiesDto = append(abilitiesDto, dto.CandidateAbility{
						ID: a.AbilityID, Name: util.FormatName(a.AbilityName, false),
					})
				}
			}
		}
	} else {
		for _, a := range abilities {
			if a.AbilityGen <= gen && genRemovedValue(a.GenRemoved) == 0 {
				abilitiesDto = append(abilitiesDto, dto.CandidateAbility{
					ID: a.AbilityID, Name: util.FormatName(a.AbilityName, false),
				})
			}
		}
	}

	seenMoves := map[int32]bool{}
	movesDto := []dto.CandidateMove{}
	for _, m := range moveRows {
		if versionGroupName != "national" {
			if m.Version == nil || *m.Version != versionGroupName {
				continue
			}
		}
		if seenMoves[m.MoveID] {
			continue
		}
		seenMoves[m.MoveID] = true
		movesDto = append(movesDto, dto.CandidateMove{
			ID: m.MoveID, Name: util.FormatName(m.Name, false),
			Type: pokeenum.ToDisplay(m.Type), MoveClass: pokeenum.ToDisplay(m.Class),
		})
	}

	return dto.TeamCandidate{
		ID: cand.ID, Name: cand.Name, Type1: cand.Type1, Type2: cand.Type2,
		Gen: cand.Gen, GenderRate: cand.GenderRate,
		Abilities: abilitiesDto, Moves: movesDto,
	}, nil
}

func nilIfEmptyPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	display := pokeenum.ToDisplay(*s)
	return &display
}

func genRemovedValue(g *int32) int32 {
	if g == nil {
		return 0
	}
	return *g
}
