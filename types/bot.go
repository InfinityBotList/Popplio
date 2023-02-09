package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateBot struct {
	BotID            string   `db:"bot_id" json:"bot_id" validate:"required,numeric" msg:"Bot ID must be numeric"`                                                                                                                                       // impld
	ClientID         string   `db:"client_id" json:"client_id" validate:"required,numeric" msg:"Client ID must be numeric"`                                                                                                                              // impld
	Short            string   `db:"short" json:"short" validate:"required,min=30,max=150" msg:"Short description must be between 30 and 150 characters"`                                                                                                 // impld
	Long             string   `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                                                                                                                 // impld
	Prefix           string   `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`                                                                                                              // impld
	AdditionalOwners []string `db:"additional_owners" json:"additional_owners" validate:"required,unique,max=7,dive,numeric" msg:"You can only have a maximum of 7 additional owners" amsg:"Each additional owner must be a valid snowflake and unique"` // impld
	Invite           string   `db:"invite" json:"invite" validate:"required,url" msg:"Invite is required and must be a valid URL"`                                                                                                                       // impld
	Banner           *string  `db:"banner" json:"banner" validate:"omitempty,url" msg:"Background must be a valid URL"`                                                                                                                                  // impld
	Library          string   `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`                                                                                                           // impld
	ExtraLinks       []Link   `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                                                                                                                   // Impld
	Tags             []string `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=20,alpha,notblank,nonvulgar,nospaces" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 20 characters and alphabetic"`
	NSFW             bool     `db:"nsfw" json:"nsfw"`
	CrossAdd         bool     `db:"cross_add" json:"cross_add"`
	StaffNote        *string  `db:"approval_note" json:"staff_note" validate:"omitempty,max=512" msg:"Staff note must be less than 512 characters if sent"` // impld
}

type IndexBot struct {
	BotID       string       `db:"bot_id" json:"bot_id"`
	User        *DiscordUser `db:"-" json:"user"`
	Short       string       `db:"short" json:"short"`
	Type        string       `db:"type" json:"type"`
	Vanity      string       `db:"vanity" json:"vanity"`
	Votes       int          `db:"votes" json:"votes"`
	Shards      int          `db:"shards" json:"shards"`
	Library     string       `db:"library" json:"library"`
	InviteClick int          `db:"invite_clicks" json:"invite_clicks"`
	Servers     int          `db:"servers" json:"servers"`
	NSFW        bool         `db:"nsfw" json:"nsfw"`
	Tags        []string     `db:"tags" json:"tags"`
	Premium     bool         `db:"premium" json:"premium"`
	Views       int          `db:"clicks" json:"clicks"`
	Banner      pgtype.Text  `db:"banner" json:"banner"`
}

// For documentation purposes
type BotStatsDocs struct {
	Servers   int `json:"servers"`
	Shards    int `json:"shards"`
	UserCount int `json:"user_count"`
}

// This uses any to allow bad stats to still work
type BotStats struct {
	// Fields are ordered in the way they are searched
	// The simple servers, shards way
	Servers *any `json:"servers"`
	Shards  *any `json:"shards"`

	// Fates List uses this (server count)
	GuildCount *any `json:"guild_count"`

	// Top.gg uses this (server count)
	ServerCount *any `json:"server_count"`

	// Top.gg and Fates List uses this (shard count)
	ShardCount *any `json:"shard_count"`

	// Rovel Discord List and dlist.gg (kinda) uses this (server count)
	Count *any `json:"count"`

	// Discordbotlist.com uses this (server count)
	Guilds *any `json:"guilds"`

	Users     *any `json:"users"`
	UserCount *any `json:"user_count"`
}

// Bot represents a bot
// A bot is a Discord bot that is on the infinitybotlist.
type Bot struct {
	ITag                     pgtype.UUID        `db:"itag" json:"itag"`
	BotID                    string             `db:"bot_id" json:"bot_id"`
	ClientID                 string             `db:"client_id" json:"client_id"`
	QueueName                string             `db:"queue_name" json:"queue_name"` // Used purely by the queue system
	QueueAvatar              string             `db:"queue_avatar" json:"queue_avatar"`
	ExtraLinks               []Link             `db:"extra_links" json:"extra_links"`
	Tags                     []string           `db:"tags" json:"tags"`
	Prefix                   pgtype.Text        `db:"prefix" json:"prefix"`
	User                     *DiscordUser       `json:"user"` // Must be parsed internally
	Owner                    string             `db:"owner" json:"-"`
	MainOwner                *DiscordUser       `json:"owner"` // Must be parsed internally
	AdditionalOwners         []string           `db:"additional_owners" json:"-"`
	ResolvedAdditionalOwners []*DiscordUser     `json:"additional_owners"` // Must be parsed internally
	StaffBot                 bool               `db:"staff_bot" json:"staff_bot"`
	Short                    string             `db:"short" json:"short"`
	Long                     string             `db:"long" json:"long"`
	Library                  string             `db:"library" json:"library"`
	NSFW                     pgtype.Bool        `db:"nsfw" json:"nsfw"`
	Premium                  pgtype.Bool        `db:"premium" json:"premium"`
	Servers                  int                `db:"servers" json:"servers"`
	Shards                   int                `db:"shards" json:"shards"`
	ShardList                []int              `db:"shard_list" json:"shard_list"`
	Users                    int                `db:"users" json:"users"`
	Votes                    int                `db:"votes" json:"votes"`
	Views                    int                `db:"clicks" json:"clicks"`
	UniqueClicks             int64              `json:"unique_clicks"` // Must be parsed internally
	InviteClicks             int                `db:"invite_clicks" json:"invite_clicks"`
	Banner                   pgtype.Text        `db:"banner" json:"banner"`
	Invite                   pgtype.Text        `db:"invite" json:"invite"`
	Type                     string             `db:"type" json:"type"` // For auditing reasons, we do not filter out denied/banned bots in API
	Vanity                   string             `db:"vanity" json:"vanity"`
	ExternalSource           pgtype.Text        `db:"external_source" json:"external_source"`
	ListSource               pgtype.UUID        `db:"list_source" json:"list_source"`
	VoteBanned               bool               `db:"vote_banned" json:"vote_banned"`
	CrossAdd                 bool               `db:"cross_add" json:"cross_add"`
	StartPeriod              pgtype.Timestamptz `db:"start_premium_period" json:"start_premium_period"`
	SubPeriod                time.Duration      `db:"premium_period_length" json:"-"`
	SubPeriodParsed          Interval           `db:"-" json:"premium_period_length"` // Must be parsed internally
	CertReason               pgtype.Text        `db:"cert_reason" json:"cert_reason"`
	Announce                 bool               `db:"announce" json:"announce"`
	AnnounceMessage          pgtype.Text        `db:"announce_message" json:"announce_message"`
	Uptime                   int                `db:"uptime" json:"uptime"`
	TotalUptime              int                `db:"total_uptime" json:"total_uptime"`
	ClaimedBy                pgtype.Text        `db:"claimed_by" json:"claimed_by"`
	Note                     pgtype.Text        `db:"approval_note" json:"approval_note"`
	CreatedAt                pgtype.Timestamptz `db:"created_at" json:"created_at"`
	LastClaimed              pgtype.Timestamptz `db:"last_claimed" json:"last_claimed"`
	WebhookHMAC              bool               `db:"hmac" json:"webhook_hmac_auth"`
	QueueReason              pgtype.Text        `db:"queue_reason" json:"queue_reason"`
}

// All bots
type AllBots struct {
	Count    uint64     `json:"count"`
	PerPage  uint64     `json:"per_page"`
	Next     string     `json:"next"`
	Previous string     `json:"previous"`
	Results  []IndexBot `json:"bots"`
}

type Invite struct {
	Invite string `json:"invite"`
}

// List Stats
type ListStatsBot struct {
	BotID              string   `json:"bot_id"`
	Vanity             string   `json:"vanity"`
	Short              string   `json:"short"`
	Type               string   `json:"type"`
	MainOwnerID        string   `json:"main_owner_id"`
	AdditionalOwnerIDS []string `json:"additional_owners_ids"`
	QueueName          string   `json:"queue_name"`
}
