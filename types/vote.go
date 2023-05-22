package types

// Vote Info
type VoteInfo struct {
	Weekend  bool   `json:"is_weekend"`
	VoteTime uint16 `json:"vote_time"`
}

type UserVote struct {
	UserID       string   `json:"user_id"`
	Timestamps   []int64  `json:"ts"`
	HasVoted     bool     `json:"has_voted"`
	LastVoteTime int64    `json:"last_vote_time"`
	VoteInfo     VoteInfo `json:"vote_info"`
	PremiumBot   bool     `json:"premium_bot"`
}
