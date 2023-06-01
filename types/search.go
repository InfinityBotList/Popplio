package types

type SearchFilter struct {
	From uint32 `json:"from"`
	To   uint32 `json:"to"`
}

type TagMode string

const (
	TagModeAll TagMode = "@>"
	TagModeAny TagMode = "&&"
)

type TagFilter struct {
	Tags    []string `json:"tags"`
	TagMode TagMode  `json:"tag_mode"`
}

type SearchQuery struct {
	Query     string       `json:"query"`
	Servers   SearchFilter `json:"servers" msg:"Servers must be a valid filter"`
	Votes     SearchFilter `json:"votes" msg:"Votes must be a valid filter"`
	Shards    SearchFilter `json:"shards" msg:"Shards must be a valid filter"`
	TagFilter TagFilter    `json:"tags" msg:"Tags must be a valid filter"`
}

type SearchResponse struct {
	Bots []IndexBot `json:"bots"`
}