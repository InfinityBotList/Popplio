package types

// Vote Info
type VoteInfo struct {
	Weekend  bool   `json:"is_weekend"`
	VoteTime uint16 `json:"vote_time"`
}

type UserVote struct {
	Timestamps   []int64  `json:"ts"`
	HasVoted     bool     `json:"has_voted"`
	LastVoteTime int64    `json:"last_vote_time"`
	VoteInfo     VoteInfo `json:"vote_info"`
}

type WebhookPost struct {
	BotID  string `json:"bot_id" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
	Test   bool   `json:"test"`
	Votes  int    `json:"votes" validate:"required"`

	// Only present on test webhook API or during sends internally
	URL string `json:"url" validate:"required"`

	// Only present on test webhook API
	Token string `json:"token" validate:"required"`

	// Only present on test webhook API
	HMACAuth bool `json:"hmac_auth"`
}

type WebhookData struct {
	Votes        int          `json:"votes"`
	UserID       string       `json:"user"`
	UserObj      *DiscordUser `json:"userObj"`
	BotID        string       `json:"bot"`
	UserIDLegacy string       `json:"userID"`
	BotIDLegacy  string       `json:"botID"`
	Test         bool         `json:"test"`
	Time         int64        `json:"time"`
}