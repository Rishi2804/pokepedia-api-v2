package dto

type SearchHit struct {
	Type string `json:"type"`
	ID   int32  `json:"id"`
	Name string `json:"name"`
	Gen  int32  `json:"gen"`
}

type SearchGroup struct {
	Type  string      `json:"type"`
	Total int         `json:"total"`
	Hits  []SearchHit `json:"hits"`
}

type SearchResponse struct {
	Query    string        `json:"query"`
	Degraded bool          `json:"degraded"`
	Groups   []SearchGroup `json:"groups"`
}
