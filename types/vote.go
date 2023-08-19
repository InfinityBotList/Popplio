package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// @ci table=entity_votes
//
// Entity Vote represents a vote on an entity.
type EntityVote struct {
	ITag       pgtype.UUID      `db:"itag" json:"itag" description:"The internal ID of the entity."`
	TargetType string           `db:"target_type" json:"target_type" description:"The type of the entity that was voted on"`
	TargetID   string           `db:"target_id" json:"target_id" description:"The ID of the entity that was voted on"`
	AuthorID   string           `db:"author" json:"author" description:"The ID of the user who voted"`
	Upvote     bool             `db:"upvote" json:"upvote" description:"Whether or not the vote was an upvote"`
	Void       bool             `db:"void" json:"void" description:"Whether or not the vote was voided"`
	VoidReason pgtype.Text      `db:"void_reason" json:"void_reason" description:"The reason the vote was voided"`
	VoidedAt   pgtype.Timestamp `db:"voided_at" json:"voided_at" description:"The time the vote was voided, if it was voided"`
	CreatedAt  time.Time        `db:"created_at" json:"created_at"`
	VoteNum    int              `db:"vote_num" json:"vote_num" description:"The number of the vote (second vote of double vote will have vote_num as 2 etc.)"`
}

// Vote Info
type VoteInfo struct {
	PerUser  int    `json:"per_user" description:"The amount of votes a single vote creates on this entity"`
	VoteTime uint16 `json:"vote_time" description:"The amount of time in hours until a user can vote again"`
}

// Stores the hours, minutes and seconds until the user can vote again
type VoteWait struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

type ValidVote struct {
	Upvote    bool      `json:"upvote" description:"Whether or not the vote was an upvote"`
	CreatedAt time.Time `json:"created_at" description:"The time the vote was created"`
}

// A user vote is a struct containing basic info on a users vote
type UserVote struct {
	HasVoted   bool         `json:"has_voted" description:"Whether or not the user has voted"`
	ValidVotes []*ValidVote `json:"valid_votes" description:"Some information about a valid vote"`
	VoteInfo   *VoteInfo    `json:"vote_info" description:"Some information about the vote"`
	Wait       *VoteWait    `json:"wait" description:"The time until the user can vote again"`
}

type HCaptchaInfo struct {
	SiteKey string `json:"site_key"`
}
