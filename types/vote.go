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
	Credit     pgtype.UUID      `db:"credit_redeem" json:"credit_redeem" description:"If the vote has been redeemed for credits, what the ID of the credit redemption is"`
	Immutable  bool             `db:"immutable" json:"immutable" description:"Whether or not the vote is immutable"`
}

// Vote Info
type VoteInfo struct {
	PerUser                          int    `json:"per_user" description:"The amount of votes a single vote creates on this entity"`
	VoteTime                         uint16 `json:"vote_time" description:"The amount of time in hours until a user can vote again"`
	VoteCredits                      bool   `json:"vote_credits" description:"Whether or not the entity supports vote credits"`
	MultipleVotes                    bool   `json:"multiple_votes" description:"Whether or not the entity supports multiple votes per time interval"`
	SupportsUpvotes                  bool   `json:"supports_upvotes" description:"Whether or not the entity supports upvotes"`
	SupportsDownvotes                bool   `json:"supports_downvotes" description:"Whether or not the entity supports downvotes"`
	SupportsPartialVoteCreditsRedeem bool   `json:"supports_partial_vote_credits_redeem" description:"Whether or not the entity supports partial vote credit redemption"`
}

// Stores the hours, minutes and seconds until the user can vote again
type VoteWait struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

// A user vote is a struct containing basic info on a users vote
type UserVote struct {
	HasVoted   bool         `json:"has_voted" description:"Whether or not the user has voted for the entity. If an entity supports multiple votes, this will be true if the user has voted in the last vote time, otherwise, it will be true if the user has voted at all"`
	ValidVotes []EntityVote `json:"valid_votes" description:"A list of all non-voided votes the user has made on the entity"`
	VoteInfo   *VoteInfo    `json:"vote_info" description:"Some information about the vote"`
	Wait       *VoteWait    `json:"wait" description:"The time until the user can vote again"`
}

type HCaptchaInfo struct {
	SiteKey string `json:"site_key"`
}

// @ci table=vote_credit_tiers
//
// VoteCreditTier represents a vote credit tier.
type VoteCreditTier struct {
	ID         string    `db:"id" json:"id" description:"The ID of the vote credit tier"`
	TargetType string    `db:"target_type" json:"target_type" description:"The target type of the entity"`
	Position   int       `db:"position" json:"position" description:"The position of the vote credit tier"`
	Votes      int       `db:"votes" json:"votes" description:"The amount of votes the user needs to get this tier"`
	Cents      int       `db:"cents" json:"cents" description:"The amount of cents the user gets off in this tier"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// Represents a summary of what would happen on redeeming vote credit tiers
type VoteCreditTierRedeemSummary struct {
	Tiers        []*VoteCreditTier `json:"tiers" description:"The vote credit tiers"`
	Votes        int               `json:"votes" description:"The amount of votes the entity has"`
	SlabOverview []int             `json:"slab_overview" description:"Slab-based overview with each index, i, representing the amount of votes in Tiers[i]"`
	TotalCredits int               `json:"total_credits" description:"The total amount of credits the user would get, in cents"`
	VoteInfo     *VoteInfo         `json:"vote_info" description:"Some information about the vote"`
}

// Represents a vote credit redeem log
type EntityVoteRedeemLog struct {
	ID              pgtype.UUID `db:"id" json:"id" description:"The ID of the vote credit redeem log"`
	TargetID        string      `db:"target_id" json:"target_id" description:"The ID of the entity that was voted on"`
	TargetType      string      `db:"target_type" json:"target_type" description:"The type of the entity that was voted on"`
	Credits         int         `db:"credits" json:"credits" description:"The amount of credits redeemed"`
	RedeemedCredits int         `db:"redeemed_credits" json:"redeemed_credits" description:"The amount of credits redeemed"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
	RedeemedAt      *time.Time  `db:"redeemed_at" json:"redeemed_at" description:"The last time the credits were redeemed for a transaction, if applicable"`
}

// Summary of the entity vote redeem log
type EntityVoteRedeemLogSummary struct {
	Redeems          []*EntityVoteRedeemLog `json:"redeems" description:"The vote credit redeem logs"`
	TotalCredits     int                    `json:"total_credits" description:"The total amount of credits available"`
	AvailableCredits int                    `json:"available_credits" description:"The total amount of credits that can be redeemed"`
	RedeemedCredits  int                    `json:"redeemed_credits" description:"The total amount of credits that have been redeemed"`
}
