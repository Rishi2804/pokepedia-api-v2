package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/search"
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

// SearchService prefers Elasticsearch and falls back to a Postgres trigram
// search when it's disabled, unhealthy, or erroring.
type SearchService struct {
	es *search.Client
	q  *store.Queries
}

func NewSearchService(es *search.Client, q *store.Queries) *SearchService {
	return &SearchService{es: es, q: q}
}

func (s *SearchService) Search(ctx context.Context, query string, size int) (*dto.SearchResponse, error) {
	if s.es.Enabled() {
		res, err := s.es.Search(ctx, search.Params{Query: query, Size: size})
		if err == nil {
			return toSearchResponse(query, res), nil
		}
		if !errors.Is(err, search.ErrUnavailable) {
			return nil, err
		}
	}
	return s.searchFallback(ctx, query, size)
}

func (s *SearchService) Suggest(ctx context.Context, query string, limit int) ([]dto.SearchHit, error) {
	if s.es.Enabled() {
		hits, err := s.es.Suggest(ctx, query, limit)
		if err == nil {
			return toSearchHits(hits), nil
		}
		if !errors.Is(err, search.ErrUnavailable) {
			return nil, err
		}
	}
	return s.suggestFallback(ctx, query, limit)
}

func toSearchResponse(query string, res *search.Results) *dto.SearchResponse {
	groups := make([]dto.SearchGroup, len(res.Groups))
	for i, g := range res.Groups {
		groups[i] = dto.SearchGroup{Type: g.Type, Total: int(g.Total), Hits: toSearchHits(g.Hits)}
	}
	return &dto.SearchResponse{Query: query, Degraded: false, Groups: groups}
}

func toSearchHits(hits []search.Hit) []dto.SearchHit {
	out := make([]dto.SearchHit, len(hits))
	for i, h := range hits {
		out[i] = dto.SearchHit{Type: h.Type, ID: h.EntityID, Name: h.Name, Gen: h.Gen, Meta: h.Meta}
	}
	return out
}

// searchFallback and suggestFallback are unchanged from the pre-Elasticsearch
// version: a Postgres trigram search, always Degraded: true.

func (s *SearchService) searchFallback(ctx context.Context, query string, size int) (*dto.SearchResponse, error) {
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
			g.Hits = append(g.Hits, toFallbackHit(r))
		}
	}

	groups := make([]dto.SearchGroup, len(order))
	for i, t := range order {
		groups[i] = *byType[t]
	}

	return &dto.SearchResponse{Query: query, Degraded: true, Groups: groups}, nil
}

func (s *SearchService) suggestFallback(ctx context.Context, query string, limit int) ([]dto.SearchHit, error) {
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
		hits[i] = toFallbackHit(r)
	}
	return hits, nil
}

func toFallbackHit(r store.SearchNamesFallbackRow) dto.SearchHit {
	return dto.SearchHit{
		Type: r.Type,
		ID:   r.ID,
		Name: util.FormatName(r.Name, r.Type == "pokemon"),
		Gen:  r.Gen,
	}
}
