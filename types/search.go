package types

type SearchFilter struct {
	From int64 `json:"from" `
	To   int64 `json:"to"`
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
	Query        string       `json:"query"`
	TargetTypes  []string     `json:"target_types"` // Defaults to 'bot' if unset
	Servers      SearchFilter `json:"servers" msg:"Servers must be a valid filter"`
	Votes        SearchFilter `json:"votes" msg:"Votes must be a valid filter"`
	Shards       SearchFilter `json:"shards" msg:"Shards must be a valid filter"`
	TotalMembers SearchFilter `json:"total_members" msg:"Total members must be a valid filter"`
	TagFilter    TagFilter    `json:"tags" msg:"Tags must be a valid filter"`
}

type SearchResponse struct {
	TargetTypes []string      `json:"target_types"`
	Bots        []IndexBot    `json:"bots,omitempty"`
	Servers     []IndexServer `json:"servers,omitempty"`
}
