package service

import (
	"context"
	"strings"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

// fallbackScanLimit is well above any realistic match count (the corpus is
// 2,383 rows), so grouped Total stays an exact count, not a truncated one.
const fallbackScanLimit = 500

const (
	defaultGroupSize    = 5
	defaultSuggestLimit = 8
)

// SearchService has no Elasticsearch client yet — that lands in a later
// commit. Every call goes through the Postgres fallback for now.
type SearchService struct {
	q *store.Queries
}

func NewSearchService(q *store.Queries) *SearchService {
	return &SearchService{q: q}
}

func (s *SearchService) Search(ctx context.Context, query string, size int) (*dto.SearchResponse, error) {
	if len(strings.TrimSpace(query)) < 2 {
		return &dto.SearchResponse{Query: query, Groups: []dto.SearchGroup{}}, nil
	}
	if size <= 0 {
		size = defaultGroupSize
	}

	rows, err := s.q.SearchNamesFallback(ctx, store.SearchNamesFallbackParams{
		Term: util.Slugify(query),
		Lim:  fallbackScanLimit,
	})
	if err != nil {
		return nil, err
	}

	// rows arrive ORDER BY score DESC, so first-seen order of types is
	// already "best hit per group" order.
	var order []string
	byType := map[string]*dto.SearchGroup{}
	for _, r := range rows {
		g, ok := byType[r.Type]
		if !ok {
			g = &dto.SearchGroup{Type: r.Type}
			byType[r.Type] = g
			order = append(order, r.Type)
		}
		g.Total++
		if len(g.Hits) < size {
			g.Hits = append(g.Hits, toSearchHit(r))
		}
	}

	groups := make([]dto.SearchGroup, len(order))
	for i, t := range order {
		groups[i] = *byType[t]
	}

	return &dto.SearchResponse{Query: query, Degraded: true, Groups: groups}, nil
}

func (s *SearchService) Suggest(ctx context.Context, query string, limit int) ([]dto.SearchHit, error) {
	if len(strings.TrimSpace(query)) < 2 {
		return []dto.SearchHit{}, nil
	}
	if limit <= 0 {
		limit = defaultSuggestLimit
	}

	rows, err := s.q.SearchNamesFallback(ctx, store.SearchNamesFallbackParams{
		Term: util.Slugify(query),
		Lim:  int32(limit),
	})
	if err != nil {
		return nil, err
	}

	hits := make([]dto.SearchHit, len(rows))
	for i, r := range rows {
		hits[i] = toSearchHit(r)
	}
	return hits, nil
}

func toSearchHit(r store.SearchNamesFallbackRow) dto.SearchHit {
	return dto.SearchHit{
		Type: r.Type,
		ID:   r.ID,
		Name: util.FormatName(r.Name, r.Type == "pokemon"),
		Gen:  r.Gen,
	}
}
