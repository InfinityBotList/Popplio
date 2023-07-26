package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// @ci table=entity_votes
//
// Entity Vote represents a vote on an entity.
type EntityVote struct {
	ITag       pgtype.UUID `db:"itag" json:"itag" description:"The internal ID of the entity."`
	TargetType string      `db:"target_type" json:"target_type"`
	TargetID   string      `db:"target_id" json:"target_id"`
	AuthorID   string      `db:"author" json:"author"`
	Upvote     bool        `db:"upvote" json:"upvote"`
	Void       bool        `db:"void" json:"void"`
	VoidReason string      `db:"void_reason" json:"void_reason"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
}

// Vote Info
type VoteInfo struct {
	DoubleVotes bool   `json:"double_votes"`
	VoteTime    uint16 `json:"vote_time"`
}

// Stores the hours, minutes and seconds until the user can vote again
type VoteWait struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

// A user vote is a struct containing basic info on a users vote
type UserVote struct {
	HasVoted   bool        `json:"has_voted"`
	ValidVotes []time.Time `json:"valid_votes"`
	VoteInfo   *VoteInfo   `json:"vote_info"`
	Wait       *VoteWait   `json:"wait"`
}

type HCaptchaInfo struct {
	SiteKey string `json:"site_key"`
}
