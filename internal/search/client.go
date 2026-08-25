package search

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"

	essearch "github.com/elastic/go-elasticsearch/v9/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
)

// min_gram is 2, so shorter queries can't match anything real.
const minQueryLen = 2

// Mirrors mapping.json's max_result_window; ES would 400 past this anyway.
const maxWindow = 1000

func tooShort(q string) bool {
	return len(strings.TrimSpace(q)) < minQueryLen
}

// Suggest returns a small, name-ranked list for the header typeahead.
func (c *Client) Suggest(ctx context.Context, q string, size int) ([]Hit, error) {
	if tooShort(q) {
		return []Hit{}, nil
	}
	if !c.Enabled() || !c.br.Allow() {
		return nil, ErrUnavailable
	}

	body, err := json.Marshal(buildSuggest(q, size))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.opTO)
	defer cancel()

	res, err := c.es.Search().Index(c.index).Raw(bytes.NewReader(body)).Do(ctx)
	if err != nil {
		c.onErr(err)
		return nil, ErrUnavailable
	}
	c.br.Success()
	c.queries.Add(1)

	hits := make([]Hit, 0, len(res.Hits.Hits))
	for _, h := range res.Hits.Hits {
		hit, err := decodeHit(h)
		if err != nil {
			continue
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

// Search runs the grouped results-page query.
func (c *Client) Search(ctx context.Context, p Params) (*Results, error) {
	if tooShort(p.Query) {
		return &Results{Groups: []Group{}}, nil
	}
	if !c.Enabled() || !c.br.Allow() {
		return nil, ErrUnavailable
	}

	body, err := json.Marshal(buildSearch(p.Query, p.Size))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.opTO)
	defer cancel()

	res, err := c.es.Search().Index(c.index).Raw(bytes.NewReader(body)).Do(ctx)
	if err != nil {
		c.onErr(err)
		return nil, ErrUnavailable
	}
	c.br.Success()
	c.queries.Add(1)

	return decodeGroupedResults(res)
}

// SearchByType is the "show all" drilldown for one type.
func (c *Client) SearchByType(ctx context.Context, p Params) (*Results, error) {
	if tooShort(p.Query) {
		return &Results{Groups: []Group{}}, nil
	}
	if p.From+p.Size > maxWindow {
		return nil, ErrUnavailable
	}
	if !c.Enabled() || !c.br.Allow() {
		return nil, ErrUnavailable
	}

	body, err := json.Marshal(buildDrilldown(p.Query, p.Type, p.From, p.Size))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.opTO)
	defer cancel()

	res, err := c.es.Search().Index(c.index).Raw(bytes.NewReader(body)).Do(ctx)
	if err != nil {
		c.onErr(err)
		return nil, ErrUnavailable
	}
	c.br.Success()
	c.queries.Add(1)

	group := Group{Type: p.Type, Hits: []Hit{}}
	if res.Hits.Total != nil {
		group.Total = res.Hits.Total.Value
	}
	for _, h := range res.Hits.Hits {
		hit, err := decodeHit(h)
		if err != nil {
			continue
		}
		group.Hits = append(group.Hits, hit)
	}

	total := group.Total
	return &Results{Total: total, Groups: []Group{group}}, nil
}

func decodeHit(h types.Hit) (Hit, error) {
	var doc Doc
	if err := json.Unmarshal(h.Source_, &doc); err != nil {
		return Hit{}, err
	}

	var score float64
	if h.Score_ != nil {
		score = float64(*h.Score_)
	}

	var highlight []string
	for _, frags := range h.Highlight {
		highlight = append(highlight, frags...)
	}

	return Hit{Doc: doc, Score: score, Highlight: highlight}, nil
}

// Aggregation responses are typed as interfaces, hence the type assertions;
// an unexpected shape degrades to empty rather than panicking.
func decodeGroupedResults(res *essearch.Response) (*Results, error) {
	out := &Results{Groups: []Group{}}
	if res.Hits.Total != nil {
		out.Total = res.Hits.Total.Value
	}

	agg, ok := res.Aggregations["by_type"].(*types.StringTermsAggregate)
	if !ok {
		return out, nil
	}
	buckets, ok := agg.Buckets.([]types.StringTermsBucket)
	if !ok {
		return out, nil
	}

	for _, b := range buckets {
		key, _ := b.Key.(string)
		g := Group{Type: key, Total: b.DocCount, Hits: []Hit{}}

		if th, ok := b.Aggregations["top"].(*types.TopHitsAggregate); ok {
			for _, h := range th.Hits.Hits {
				hit, err := decodeHit(h)
				if err != nil {
					continue
				}
				g.Hits = append(g.Hits, hit)
			}
		}
		out.Groups = append(out.Groups, g)
	}

	// Buckets arrive ordered by doc_count; re-order by best hit instead.
	sort.SliceStable(out.Groups, func(i, j int) bool {
		return bestScore(out.Groups[i]) > bestScore(out.Groups[j])
	})

	return out, nil
}

func bestScore(g Group) float64 {
	if len(g.Hits) == 0 {
		return 0
	}
	return g.Hits[0].Score
}
