package search

import "encoding/json"

// Doc is one document in the pokepedia-search index — see mapping.json.
// Meta is entity-type-specific, so it's left undecoded here; callers decode
// it once they know Type.
type Doc struct {
	Type     string          `json:"type"`
	EntityID int32           `json:"entity_id"`
	Slug     string          `json:"slug"`
	Name     string          `json:"name"`
	Gen      int32           `json:"gen"`
	Meta     json.RawMessage `json:"meta,omitempty"`
}

type Hit struct {
	Doc
	Score     float64  `json:"score"`
	Highlight []string `json:"highlight,omitempty"`
}

type Group struct {
	Type  string `json:"type"`
	Total int64  `json:"total"`
	Hits  []Hit  `json:"hits"`
}

type Results struct {
	Total  int64   `json:"total"`
	Groups []Group `json:"groups"`
}

// Params are the inputs shared by Suggest and Search.
type Params struct {
	Query string
	Size  int
	Type  string // optional, narrows to one type for a "show all" drilldown
	From  int
}
