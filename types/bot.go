package types

import (
	"time"

	"github.com/infinitybotlist/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

type IndexBot struct {
	BotID       string                `db:"bot_id" json:"bot_id" description:"The bot's ID"`
	User        *dovewing.DiscordUser `db:"-" json:"user" description:"The bot's user information"`
	Short       string                `db:"short" json:"short" description:"The bot's short description"`
	Long        string                `db:"long" json:"long" description:"The bot's long description in raw format (HTML/markdown etc. based on the bots settings)"`
	Type        string                `db:"type" json:"type" description:"The bot's type (e.g. pending/approved/certified/denied  etc.)"`
	Vanity      string                `db:"vanity" json:"vanity" description:"The bot's vanity URL if it has one, otherwise null"`
	Votes       int                   `db:"votes" json:"votes" description:"The bot's vote count"`
	Shards      int                   `db:"shards" json:"shards" description:"The bot's shard count"`
	Library     string                `db:"library" json:"library" description:"The bot's library"`
	InviteClick int                   `db:"invite_clicks" json:"invite_clicks" description:"The bot's invite click count (via users inviting the bot from IBL)"`
	Servers     int                   `db:"servers" json:"servers" description:"The bot's server count"`
	NSFW        bool                  `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW or not"`
	Tags        []string              `db:"tags" json:"tags" description:"The bot's tags (e.g. music, moderation, etc.)"`
	Premium     bool                  `db:"premium" json:"premium" description:"Whether the bot is a premium bot or not"`
	Views       int                   `db:"clicks" json:"clicks" description:"The bot's view count"`
	Banner      pgtype.Text           `db:"banner" json:"banner" description:"The bot's banner URL if it has one, otherwise null"`
}

// For documentation purposes
type BotStatsDocs struct {
	Servers   int   `json:"servers" description:"The server count"`
	Shards    int   `json:"shards" description:"The shard count"`
	ShardList []int `json:"shard_list" description:"The shard list"`
	UserCount int   `json:"user_count" description:"The user count (not used in webpage)"`
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

	ShardList []any `json:"shard_list"`
}

// Bot represents a bot
// A bot is a Discord bot that is on the infinitybotlist.
type Bot struct {
	ITag            pgtype.UUID           `db:"itag" json:"itag" description:"The bot's internal ID. Not that useful and more a artifact of database migrations. May be removed in the future."`
	BotID           string                `db:"bot_id" json:"bot_id" description:"The bot's ID"`
	ClientID        string                `db:"client_id" json:"client_id" description:"The bot's associated client ID validated using that top-secret Oauth2 API! Used in anti-abuse measures."`
	QueueName       string                `db:"queue_name" json:"queue_name"` // Used purely by the queue system
	QueueAvatar     string                `db:"queue_avatar" json:"queue_avatar" description:"The bot's queue name if it has one, otherwise null"`
	ExtraLinks      []Link                `db:"extra_links" json:"extra_links" description:"The bot's links that it wishes to advertise"`
	Tags            []string              `db:"tags" json:"tags" description:"The bot's tags (e.g. music, moderation, etc.)"`
	Prefix          pgtype.Text           `db:"prefix" json:"prefix" description:"The bot's prefix"`
	User            *dovewing.DiscordUser `json:"user" description:"The bot's user information"` // Must be parsed internally
	Owner           pgtype.Text           `db:"owner" json:"-"`
	MainOwner       *dovewing.DiscordUser `json:"owner" description:"The bot owner's user information"` // Must be parsed internally
	Short           string                `db:"short" json:"short" description:"The bot's short description"`
	Long            string                `db:"long" json:"long" description:"The bot's long description in raw format (HTML/markdown etc. based on the bots settings)"`
	Library         string                `db:"library" json:"library" description:"The bot's library"`
	NSFW            pgtype.Bool           `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW or not"`
	Premium         pgtype.Bool           `db:"premium" json:"premium" description:"Whether the bot is a premium bot or not"`
	Servers         int                   `db:"servers" json:"servers" description:"The bot's server count"`
	Shards          int                   `db:"shards" json:"shards" description:"The bot's shard count"`
	ShardList       []int                 `db:"shard_list" json:"shard_list" description:"The number of servers per shard"`
	Users           int                   `db:"users" json:"users" description:"The bot's user count"`
	Votes           int                   `db:"votes" json:"votes"`
	Views           int                   `db:"clicks" json:"clicks"`
	UniqueClicks    int64                 `json:"unique_clicks"` // Must be parsed internally
	InviteClicks    int                   `db:"invite_clicks" json:"invite_clicks"`
	Banner          pgtype.Text           `db:"banner" json:"banner"`
	Invite          pgtype.Text           `db:"invite" json:"invite"`
	Type            string                `db:"type" json:"type"` // For auditing reasons, we do not filter out denied/banned bots in API
	Vanity          string                `db:"vanity" json:"vanity"`
	ExternalSource  pgtype.Text           `db:"external_source" json:"external_source"`
	ListSource      pgtype.UUID           `db:"list_source" json:"list_source"`
	VoteBanned      bool                  `db:"vote_banned" json:"vote_banned"`
	CrossAdd        bool                  `db:"cross_add" json:"cross_add"`
	StartPeriod     pgtype.Timestamptz    `db:"start_premium_period" json:"start_premium_period"`
	SubPeriod       time.Duration         `db:"premium_period_length" json:"-"`
	SubPeriodParsed Interval              `db:"-" json:"premium_period_length"` // Must be parsed internally
	CertReason      pgtype.Text           `db:"cert_reason" json:"cert_reason"`
	Uptime          int                   `db:"uptime" json:"uptime"`
	TotalUptime     int                   `db:"total_uptime" json:"total_uptime"`
	ClaimedBy       pgtype.Text           `db:"claimed_by" json:"claimed_by"`
	Note            pgtype.Text           `db:"approval_note" json:"approval_note"`
	CreatedAt       pgtype.Timestamptz    `db:"created_at" json:"created_at"`
	LastClaimed     pgtype.Timestamptz    `db:"last_claimed" json:"last_claimed"`
	WebhooksV2      bool                  `db:"webhooks_v2" json:"webhooks_v2"`
	TeamOwnerID     pgtype.UUID           `db:"team_owner" json:"-"`
	TeamOwner       *Team                 `json:"team_owner"` // Must be parsed internally
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

// List Index
type ListIndexBot struct {
	Certified     []IndexBot     `json:"certified"`
	Premium       []IndexBot     `json:"premium"`
	MostViewed    []IndexBot     `json:"most_viewed"`
	Packs         []IndexBotPack `json:"packs"`
	RecentlyAdded []IndexBot     `json:"recently_added"`
	TopVoted      []IndexBot     `json:"top_voted"`
}
