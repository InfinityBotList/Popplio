package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
	PendingCert              pgtype.Bool        `db:"pending_cert" json:"pending_cert"`
	Servers                  int                `db:"servers" json:"servers"`
	Shards                   int                `db:"shards" json:"shards"`
	ShardList                []int              `db:"shard_list" json:"shard_list"`
	Users                    int                `db:"users" json:"users"`
	Votes                    int                `db:"votes" json:"votes"`
	Views                    int                `db:"clicks" json:"clicks"`
	UniqueClicks             int64              `json:"unique_clicks"` // Must be parsed internally
	InviteClicks             int                `db:"invite_clicks" json:"invites"`
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
