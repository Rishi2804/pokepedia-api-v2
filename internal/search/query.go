package search

import (
	"strings"

	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

// Boost ladder: exact slug > exact name > prefix > infix > fuzzy > text.
const (
	boostSlugExact  = 200
	boostNameExact  = 160
	boostNamePrefix = 12
	boostNameInfix  = 6
	boostNameFuzzy  = 4
	boostTextPhrase = 2
	boostTextTerms  = 1
	boostPopularity = 3

	// Plain AUTO is too loose at short lengths ("mew" -> "dew"/"hex").
	fuzziness = "AUTO:4,7"

	defaultSuggestSize = 8
	defaultGroupSize   = 5
)

var sourceFields = []string{"type", "entity_id", "slug", "name", "gen", "meta"}

// nameLanes are the clauses shared by every query mode.
func nameLanes(q string) []any {
	slug := util.Slugify(q)
	lower := strings.ToLower(strings.TrimSpace(q))

	return []any{
		map[string]any{"term": map[string]any{
			"slug": map[string]any{"value": slug, "boost": boostSlugExact},
		}},
		map[string]any{"term": map[string]any{
			"name.kw": map[string]any{"value": lower, "boost": boostNameExact},
		}},
		// Per-token prefix: order-independent ("venus mega" -> Mega Venusaur).
		map[string]any{"multi_match": map[string]any{
			"query":    q,
			"type":     "best_fields",
			"fields":   []string{"name.ac^8", "aliases.ac^4", "slug.ac^3"},
			"operator": "and",
			"boost":    boostNamePrefix,
		}},
		// Substring within a token: "blast" -> "Moonblast".
		map[string]any{"multi_match": map[string]any{
			"query":    q,
			"type":     "best_fields",
			"fields":   []string{"name.infix^3", "aliases.infix^2"},
			"operator": "and",
			"boost":    boostNameInfix,
		}},
		map[string]any{"multi_match": map[string]any{
			"query":          q,
			"type":           "best_fields",
			"fields":         []string{"name^10", "aliases^5", "slug.text^5"},
			"operator":       "and",
			"fuzziness":      fuzziness,
			"prefix_length":  1,
			"max_expansions": 30,
			"boost":          boostNameFuzzy,
		}},
	}
}

// textLanes add description/effect matching; not used by Suggest.
func textLanes(q string) []any {
	return []any{
		map[string]any{"multi_match": map[string]any{
			"query":  q,
			"type":   "phrase",
			"fields": []string{"effect^2", "description^2"},
			"slop":   1,
			"boost":  boostTextPhrase,
		}},
		map[string]any{"multi_match": map[string]any{
			"query":                q,
			"type":                 "cross_fields",
			"fields":               []string{"description", "effect"},
			"operator":             "or",
			"minimum_should_match": "2<70%",
			"boost":                boostTextTerms,
		}},
	}
}

// Must sit in an outer "should" — as a sibling matcher it'd match every
// document that has the field, i.e. the whole index.
func popularityLane() map[string]any {
	return map[string]any{
		"rank_feature": map[string]any{
			"field":      "popularity",
			"saturation": map[string]any{"pivot": 8},
			"boost":      boostPopularity,
		},
	}
}

func mustMatchOne(should []any) map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"minimum_should_match": 1,
			"should":               should,
		},
	}
}

// buildSuggest is the header typeahead: small, name-only.
func buildSuggest(q string, size int) map[string]any {
	if size <= 0 {
		size = defaultSuggestSize
	}
	return map[string]any{
		"size":             size,
		"track_total_hits": false,
		"_source":          map[string]any{"includes": sourceFields},
		"query": map[string]any{
			"bool": map[string]any{
				"must":   []any{mustMatchOne(nameLanes(q))},
				"should": []any{popularityLane()},
			},
		},
	}
}

// buildSearch is the grouped results page: size:0, hits live in the aggs.
func buildSearch(q string, groupSize int) map[string]any {
	if groupSize <= 0 {
		groupSize = defaultGroupSize
	}
	should := append(nameLanes(q), textLanes(q)...)

	return map[string]any{
		"size":             0,
		"track_total_hits": true,
		"query": map[string]any{
			"bool": map[string]any{
				"must":   []any{mustMatchOne(should)},
				"should": []any{popularityLane()},
			},
		},
		"aggs": map[string]any{
			"by_type": map[string]any{
				"terms": map[string]any{"field": "type", "size": 8},
				"aggs": map[string]any{
					"top": map[string]any{
						"top_hits": map[string]any{
							"size":    groupSize,
							"_source": map[string]any{"includes": sourceFields},
							"highlight": map[string]any{
								"pre_tags":  []string{"<em>"},
								"post_tags": []string{"</em>"},
								"fields": map[string]any{
									"description": map[string]any{"fragment_size": 140, "number_of_fragments": 1},
									"effect":      map[string]any{"fragment_size": 140, "number_of_fragments": 1},
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildDrilldown is the "show all" expander: paginated, no aggregation.
func buildDrilldown(q, entityType string, from, size int) map[string]any {
	if size <= 0 {
		size = defaultGroupSize
	}
	should := append(nameLanes(q), textLanes(q)...)

	return map[string]any{
		"from":             from,
		"size":             size,
		"track_total_hits": true,
		"_source":          map[string]any{"includes": sourceFields},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{map[string]any{"term": map[string]any{"type": entityType}}},
				"must":   []any{mustMatchOne(should)},
				"should": []any{popularityLane()},
			},
		},
	}
}
