package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

type IndexBot struct {
	BotID       string                `db:"bot_id" json:"bot_id" description:"The bot's ID"`
	User        *dovewing.DiscordUser `db:"-" json:"user" description:"The bot's user information"`
	Short       string                `db:"short" json:"short" description:"The bot's short description"`
	Long        string                `db:"long" json:"long" description:"The bot's long description in raw format (HTML/markdown etc. based on the bots settings)"`
	Type        string                `db:"type" json:"type" description:"The bot's type (e.g. pending/approved/certified/denied etc.)"`
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

type BotStats struct {
	Servers   uint64   `json:"servers" description:"The server count"`
	Shards    uint64   `json:"shards" description:"The shard count"`
	Users     uint64   `json:"users" description:"The user count (not used in webpage)"`
	ShardList []uint64 `json:"shard_list" description:"The shard list"`
}

// Bot represents a bot.
// A bot is a Discord bot that is on the infinitybotlist.
type Bot struct {
	ITag                      pgtype.UUID           `db:"itag" json:"itag" description:"The bot's internal ID. An artifact of database migrations."`
	BotID                     string                `db:"bot_id" json:"bot_id" description:"The bot's ID"`
	ClientID                  string                `db:"client_id" json:"client_id" description:"The bot's associated client ID validated using that top-secret Oauth2 API! Used in anti-abuse measures."`
	QueueName                 string                `db:"queue_name" json:"queue_name"` // Used purely by the queue system
	QueueAvatar               string                `db:"queue_avatar" json:"queue_avatar" description:"The bot's queue name if it has one, otherwise null"`
	ExtraLinks                []Link                `db:"extra_links" json:"extra_links" description:"The bot's links that it wishes to advertise"`
	Tags                      []string              `db:"tags" json:"tags" description:"The bot's tags (e.g. music, moderation, etc.)"`
	Prefix                    string                `db:"prefix" json:"prefix" description:"The bot's prefix"`
	User                      *dovewing.DiscordUser `json:"user" description:"The bot's user information"` // Must be parsed internally
	Owner                     pgtype.Text           `db:"owner" json:"-"`
	MainOwner                 *dovewing.DiscordUser `json:"owner" description:"The bot owner's user information. If in a team, this will be null and team_owner will instead be set"` // Must be parsed internally
	Short                     string                `db:"short" json:"short" description:"The bot's short description"`
	Long                      string                `db:"long" json:"long" description:"The bot's long description in raw format (HTML/markdown etc. based on the bots settings)"`
	Library                   string                `db:"library" json:"library" description:"The bot's library"`
	NSFW                      bool                  `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW or not"`
	Premium                   bool                  `db:"premium" json:"premium" description:"Whether the bot is a premium bot or not"`
	Servers                   int                   `db:"servers" json:"servers" description:"The bot's server count"`
	Shards                    int                   `db:"shards" json:"shards" description:"The bot's shard count"`
	ShardList                 []int                 `db:"shard_list" json:"shard_list" description:"The number of servers per shard"`
	Users                     int                   `db:"users" json:"users" description:"The bot's user count"`
	Votes                     int                   `db:"votes" json:"votes" description:"The bot's vote count"`
	Views                     int                   `db:"clicks" json:"clicks" description:"The bot's total click count"`
	UniqueClicks              int64                 `json:"unique_clicks" description:"The bot's unique click count based on SHA256 hashed IPs"` // Must be parsed internally
	InviteClicks              int                   `db:"invite_clicks" json:"invite_clicks" description:"The bot's invite click count (via users inviting the bot from IBL)"`
	Banner                    pgtype.Text           `db:"banner" json:"banner" description:"The bot's banner URL if it has one, otherwise null"`
	Invite                    string                `db:"invite" json:"invite" description:"The bot's invite URL. Must be present"`
	Type                      string                `db:"type" json:"type" description:"The bot's type (e.g. pending/approved/certified/denied etc.). Note that we do not filter out denied/banned bots in API"`
	Vanity                    string                `db:"vanity" json:"vanity" description:"The bot's vanity URL if it has one, otherwise null"`
	VoteBanned                bool                  `db:"vote_banned" json:"vote_banned" description:"Whether the bot is vote banned or not"`
	StartPeriod               pgtype.Timestamptz    `db:"start_premium_period" json:"start_premium_period"`
	PremiumPeriodLength       time.Duration         `db:"premium_period_length" json:"-"`
	PremiumPeriodLengthParsed Interval              `db:"-" json:"premium_period_length" description:"The period of premium for the bot"` // Must be parsed internally
	CertReason                pgtype.Text           `db:"cert_reason" json:"cert_reason" description:"The reason for the bot being certified"`
	Uptime                    int                   `db:"uptime" json:"uptime" description:"The bot's total number of successful uptime checks"`
	TotalUptime               int                   `db:"total_uptime" json:"total_uptime" description:"The bot's total number of uptime checks"`
	ClaimedBy                 pgtype.Text           `db:"claimed_by" json:"claimed_by" description:"The user who claimed the bot"`
	Note                      pgtype.Text           `db:"approval_note" json:"approval_note" description:"The note for the bot's approval"`
	CreatedAt                 pgtype.Timestamptz    `db:"created_at" json:"created_at" description:"The bot's creation date"`
	LastClaimed               pgtype.Timestamptz    `db:"last_claimed" json:"last_claimed" description:"The bot's last claimed date"`
	WebhooksV2                bool                  `db:"webhooks_v2" json:"webhooks_v2" description:"Whether the bot is using webhooks v2 or not"`
	TeamOwnerID               pgtype.UUID           `db:"team_owner" json:"-"`
	TeamOwner                 *Team                 `json:"team_owner" description:"If the bot is in a team, who owns the bot. If not in a team, this will be null and owner will instead be set"` // Must be parsed internally
	CaptchaOptOut             bool                  `db:"captcha_opt_out" json:"captcha_opt_out" description:"Whether the bot should have captchas shown if the user has captcha_sponsor_enabled"`
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
