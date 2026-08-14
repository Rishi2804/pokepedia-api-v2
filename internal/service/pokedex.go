package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

// ErrVersionNotFound signals an unrecognised version group name, so handlers can
// answer 400 rather than conflating it with a missing row or a DB failure.
var ErrVersionNotFound = errors.New("version group not found")

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
					Type2: displayPtr(r.Type2),
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
		entries := []dto.PokedexEntry{}
		for _, r := range rows {
			entries = append(entries, dto.PokedexEntry{
				DexNumber: r.Num, SpeciesID: r.SpeciesID, PokemonID: r.ID,
				Name: formatNullableName(r.Name), Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1),
				Type2: displayPtr(r.Type2),
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

		candidates := make([]dto.TeamCandidateSummary, len(rows))
		for i, r := range rows {
			candidates[i] = dto.TeamCandidateSummary{
				ID: r.ID, Name: util.FormatName(r.Name, true), Slug: r.Name, Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1), Type2: displayPtr(r.Type2), GenderRate: r.GenderRate,
			}
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

		candidates := make([]dto.TeamCandidateSummary, len(rows))
		for i, r := range rows {
			seen[r.ID] = true
			candidates[i] = dto.TeamCandidateSummary{
				ID: r.ID, Name: util.FormatName(r.Name, true), Slug: r.Name, Gen: r.Gen,
				Type1: pokeenum.ToDisplay(r.Type1), Type2: displayPtr(r.Type2), GenderRate: r.GenderRate,
			}
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

	national := make([]dto.TeamCandidateSummary, 0, len(nationalRows))
	for _, r := range nationalRows {
		if seen[r.ID] {
			continue
		}
		national = append(national, dto.TeamCandidateSummary{
			ID: r.ID, Name: util.FormatName(r.Name, true), Slug: r.Name, Gen: r.Gen,
			Type1: pokeenum.ToDisplay(r.Type1), Type2: displayPtr(r.Type2), GenderRate: r.GenderRate,
		})
	}

	if len(national) > 0 {
		result = append(result, dto.TeamBuildingGroup{
			ListName: "National", Pokemon: national,
		})
	}

	return result, nil
}

// GetTeamCandidate returns the single candidate GetTeamCandidates would have placed
// in one of its groups. The Pokemon must belong to the version group — enforced in
// SQL — so an id absent from that game yields pgx.ErrNoRows, same as an unknown id.
func (s *PokedexService) GetTeamCandidate(ctx context.Context, versionName string, id int32) (dto.TeamCandidate, error) {
	vg, err := pokeenum.GetVersionGroup(versionName)
	if err != nil {
		return dto.TeamCandidate{}, fmt.Errorf("%w: %s", ErrVersionNotFound, versionName)
	}

	var plain dto.TeamCandidateSummary
	var stats dto.Stats
	if versionName == "national" {
		row, err := s.q.GetTeamCandidateNationalByID(ctx, id)
		if err != nil {
			return dto.TeamCandidate{}, err
		}
		plain = dto.TeamCandidateSummary{
			ID: row.ID, Name: util.FormatName(row.Name, true), Slug: row.Name, Gen: row.Gen,
			Type1: pokeenum.ToDisplay(row.Type1), Type2: displayPtr(row.Type2), GenderRate: row.GenderRate,
		}
		stats = dto.Stats{
			HP: row.Hp, Atk: row.Atk, Def: row.Def,
			SpAtk: row.Spatk, SpDef: row.Spdef, Speed: row.Speed, BST: row.Bst,
		}
	} else {
		row, err := s.q.GetTeamCandidateByID(ctx, store.GetTeamCandidateByIDParams{
			Gen:       vg.Gen,
			PokemonID: id,
			Regions:   vg.Regions,
			Version:   versionName,
		})
		if err != nil {
			return dto.TeamCandidate{}, err
		}
		plain = dto.TeamCandidateSummary{
			ID: row.ID, Name: util.FormatName(row.Name, true), Slug: row.Name, Gen: row.Gen,
			Type1: pokeenum.ToDisplay(row.Type1), Type2: displayPtr(row.Type2), GenderRate: row.GenderRate,
		}
		stats = dto.Stats{
			HP: row.Hp, Atk: row.Atk, Def: row.Def,
			SpAtk: row.Spatk, SpDef: row.Spdef, Speed: row.Speed, BST: row.Bst,
		}
	}

	abilitiesByID, movesByID, err := s.batchCandidateDetails(ctx, []int32{id}, versionName)
	if err != nil {
		return dto.TeamCandidate{}, err
	}

	return buildCandidateDetail(plain, stats, abilitiesByID[id], movesByID[id], vg.Gen), nil
}

// batchCandidateDetails fetches abilities and moves for every id in one round
// trip each, grouped by pokemon_id. The version filter that buildCandidateDetail
// used to apply in Go is now expressed as a choice between two SQL queries.
func (s *PokedexService) batchCandidateDetails(ctx context.Context, ids []int32, versionGroupName string) (
	map[int32][]store.GetCandidateAbilitiesByIDsRow,
	map[int32][]store.GetCandidateMovesByIDsRow,
	error,
) {
	abilities, err := s.q.GetCandidateAbilitiesByIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	var moves []store.GetCandidateMovesByIDsRow
	if versionGroupName == "national" {
		moves, err = s.q.GetCandidateMovesByIDs(ctx, ids)
		if err != nil {
			return nil, nil, err
		}
	} else {
		versioned, err := s.q.GetCandidateMovesByIDsAndVersion(ctx, store.GetCandidateMovesByIDsAndVersionParams{
			PokemonIds: ids,
			Version:    store.Group(versionGroupName),
		})
		if err != nil {
			return nil, nil, err
		}
		moves = make([]store.GetCandidateMovesByIDsRow, len(versioned))
		for i, m := range versioned {
			moves[i] = store.GetCandidateMovesByIDsRow(m)
		}
	}

	abilitiesByID := map[int32][]store.GetCandidateAbilitiesByIDsRow{}
	for _, a := range abilities {
		abilitiesByID[a.PokemonID] = append(abilitiesByID[a.PokemonID], a)
	}
	movesByID := map[int32][]store.GetCandidateMovesByIDsRow{}
	for _, m := range moves {
		movesByID[m.PokemonID] = append(movesByID[m.PokemonID], m)
	}

	return abilitiesByID, movesByID, nil
}

// buildCandidateDetail expands a list-shaped summary into the full single-candidate
// record. It is pure: abilities/moves are pre-fetched by batchCandidateDetails. Only
// the single-candidate endpoint reaches here now — the list endpoint returns
// summaries and never pays for the ability/move lookups.
func buildCandidateDetail(cand dto.TeamCandidateSummary, stats dto.Stats, abilities []store.GetCandidateAbilitiesByIDsRow, moveRows []store.GetCandidateMovesByIDsRow, gen int32) dto.TeamCandidate {
	abilitiesDto := []dto.CandidateAbility{}

	var matched *store.GetCandidateAbilitiesByIDsRow
	for i := range abilities {
		if abilities[i].GenRemoved != nil {
			matched = &abilities[i]
			break
		}
	}

	if matched != nil && gen <= *matched.GenRemoved {
		hiddenRemoved := false
		for _, a := range abilities {
			if a.GenRemoved != nil && a.Hidden {
				hiddenRemoved = true
				break
			}
		}
		if hiddenRemoved {
			for _, a := range abilities {
				if a.GenRemoved == nil && a.Hidden {
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
			if a.AbilityGen <= gen && a.GenRemoved == nil {
				abilitiesDto = append(abilitiesDto, dto.CandidateAbility{
					ID: a.AbilityID, Name: util.FormatName(a.AbilityName, false),
				})
			}
		}
	}

	seenMoves := map[int32]bool{}
	movesDto := []dto.CandidateMove{}
	for _, m := range moveRows {
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
		TeamCandidateSummary: cand,
		Stats:                stats,
		Abilities:            abilitiesDto,
		Moves:                movesDto,
	}
}

func displayPtr(s *string) *string {
	if s == nil {
		return nil
	}
	display := pokeenum.ToDisplay(*s)
	return &display
}
