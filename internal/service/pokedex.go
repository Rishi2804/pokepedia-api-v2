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
		rows, err := s.q.GetTeamCandidatesNationalByVersionGroup(ctx, store.GetTeamCandidatesNationalByVersionGroupParams{
			Gen:     vg.Gen,
			Version: store.Group(versionName),
		})
		if err != nil {
			return nil, err
		}

		candidates := []dto.TeamCandidate{}
		for _, r := range rows {
			candidates = append(candidates, dto.TeamCandidate{
				ID: r.ID, Name: util.FormatName(r.Name, true), Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1), Type2: nilIfEmptyPtr(r.Type2), GenderRate: r.GenderRate,
			})
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
			candidates = append(candidates, dto.TeamCandidate{
				ID: r.ID, Name: util.FormatName(r.Name, true), Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1), Type2: nilIfEmptyPtr(r.Type2), GenderRate: r.GenderRate,
			})
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
		national = append(national, dto.TeamCandidate{
			ID: r.ID, Name: util.FormatName(r.Name, true), Gen: r.Gen,
			Type1: pokeenum.ToDisplay(r.Type1), Type2: nilIfEmptyPtr(r.Type2), GenderRate: r.GenderRate,
		})
	}

	if len(national) > 0 {
		result = append(result, dto.TeamBuildingGroup{
			ListName: "National", Pokemon: national,
		})
	}

	return result, nil
}

func nilIfEmptyPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	display := pokeenum.ToDisplay(*s)
	return &display
}
