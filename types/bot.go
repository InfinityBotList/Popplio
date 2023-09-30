package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type BotFlags string

const ()

// @ci table=bots, unfilled=1
//
// Represents a 'index bot' (a small subset of the bot object for use in cards etc.)
type IndexBot struct {
	BotID       string                  `db:"bot_id" json:"bot_id" description:"The bot's ID"`
	User        *dovetypes.PlatformUser `db:"-" json:"user" description:"The bot's user information" ci:"internal"` // Must be parsed internally
	Short       string                  `db:"short" json:"short" description:"The bot's short description"`
	Type        string                  `db:"type" json:"type" description:"The bot's type (e.g. pending/approved/certified/denied etc.)"`
	VanityRef   pgtype.UUID             `db:"vanity_ref" json:"vanity_ref" description:"The corresponding vanities itag, this also works to ensure that all bots have an associated vanity"`
	Vanity      string                  `db:"-" json:"vanity" description:"The bot's vanity URL" ci:"internal"` // Must be parsed internally
	Votes       int                     `db:"votes" json:"votes" description:"The bot's vote count"`
	Shards      int                     `db:"shards" json:"shards" description:"The bot's shard count"`
	Library     string                  `db:"library" json:"library" description:"The bot's library"`
	InviteClick int                     `db:"invite_clicks" json:"invite_clicks" description:"The bot's invite click count (via users inviting the bot from IBL)"`
	Clicks      int                     `db:"clicks" json:"clicks" description:"The bot's view count"`
	Servers     int                     `db:"servers" json:"servers" description:"The bot's server count"`
	NSFW        bool                    `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW or not"`
	Tags        []string                `db:"tags" json:"tags" description:"The bot's tags (e.g. music, moderation, etc.)"`
	Premium     bool                    `db:"premium" json:"premium" description:"Whether the bot is a premium bot or not"`
	HasBanner   bool                    `db:"has_banner" json:"has_banner" description:"Whether the bot has a banner or not. If it does, then it will be accessible from $cdnUrl/banners/bots/$bot_id.webp"`
}

type BotStats struct {
	Servers   uint64   `json:"servers" description:"The server count"`
	Shards    uint64   `json:"shards" description:"The shard count"`
	Users     uint64   `json:"users" description:"The user count (not used in webpage)"`
	ShardList []uint64 `json:"shard_list" description:"The shard list"`
}

// @ci table=bots
//
// Bot represents a bot.
type Bot struct {
	ITag                pgtype.UUID             `db:"itag" json:"itag" description:"The bot's internal ID. An artifact of database migrations."`
	BotID               string                  `db:"bot_id" json:"bot_id" description:"The bot's ID"`
	ClientID            string                  `db:"client_id" json:"client_id" description:"The bot's associated client ID validated using that top-secret Oauth2 API! Used in anti-abuse measures."`
	ExtraLinks          []Link                  `db:"extra_links" json:"extra_links" description:"The bot's links that it wishes to advertise"`
	Tags                []string                `db:"tags" json:"tags" description:"The bot's tags (e.g. music, moderation, etc.)"`
	Flags               []BotFlags              `db:"flags" json:"flags" description:"The bot's flags"`
	Prefix              string                  `db:"prefix" json:"prefix" description:"The bot's prefix"`
	User                *dovetypes.PlatformUser `db:"-" json:"user" description:"The bot's user information" ci:"internal"` // Must be parsed internally
	Owner               pgtype.Text             `db:"owner" json:"-"`
	MainOwner           *dovetypes.PlatformUser `db:"-" json:"owner" description:"The bot owner's user information. If in a team, this will be null and team_owner will instead be set" ci:"internal"` // Must be parsed internally
	Short               string                  `db:"short" json:"short" description:"The bot's short description"`
	Long                string                  `db:"long" json:"long,omitempty" description:"The bot's long description in raw format (HTML/markdown etc. based on the bots settings)."`
	Library             string                  `db:"library" json:"library" description:"The bot's library"`
	NSFW                bool                    `db:"nsfw" json:"nsfw" description:"Whether the bot is NSFW or not"`
	Premium             bool                    `db:"premium" json:"premium" description:"Whether the bot is a premium bot or not"`
	LastStatsPost       pgtype.Timestamptz      `db:"last_stats_post" json:"last_stats_post" description:"The list time the bot posted stats to the list. Null if never posted"`
	Servers             int                     `db:"servers" json:"servers" description:"The bot's server count"`
	Shards              int                     `db:"shards" json:"shards" description:"The bot's shard count"`
	ShardList           []int                   `db:"shard_list" json:"shard_list" description:"The number of servers per shard"`
	Users               int                     `db:"users" json:"users" description:"The bot's user count"`
	Votes               int                     `db:"votes" json:"votes" description:"The bot's vote count"`
	Clicks              int                     `db:"clicks" json:"clicks" description:"The bot's total click count"`
	UniqueClicks        int64                   `db:"-" json:"unique_clicks" description:"The bot's unique click count based on SHA256 hashed IPs" ci:"internal"` // Must be parsed internally
	InviteClicks        int                     `db:"invite_clicks" json:"invite_clicks" description:"The bot's invite click count (via users inviting the bot from IBL)"`
	HasBanner           bool                    `db:"has_banner" json:"has_banner" description:"Whether the bot has a banner or not. If it does, then it will be accessible from $cdnUrl/banners/bots/$bot_id.webp"`
	Invite              string                  `db:"invite" json:"invite" description:"The bot's invite URL. Must be present"`
	Type                string                  `db:"type" json:"type" description:"The bot's type (e.g. pending/approved/certified/denied etc.). Note that we do not filter out denied/banned bots in API"`
	VanityRef           pgtype.UUID             `db:"vanity_ref" json:"vanity_ref" description:"The corresponding vanities itag, this also works to ensure that all bots have an associated vanity"`
	Vanity              string                  `db:"-" json:"vanity" description:"The bot's vanity URL" ci:"internal"` // Must be parsed internally
	VoteBanned          bool                    `db:"vote_banned" json:"vote_banned" description:"Whether the bot is vote banned or not"`
	StartPeriod         pgtype.Timestamptz      `db:"start_premium_period" json:"start_premium_period"`
	PremiumPeriodLength time.Duration           `db:"premium_period_length" json:"premium_period_length" description:"The period of premium for the bot in nanoseconds"`
	CertReason          pgtype.Text             `db:"cert_reason" json:"cert_reason" description:"The reason for the bot being certified"`
	Uptime              int                     `db:"uptime" json:"uptime" description:"The bot's total number of successful uptime checks"`
	TotalUptime         int                     `db:"total_uptime" json:"total_uptime" description:"The bot's total number of uptime checks"`
	UptimeLastChecked   pgtype.Timestamptz      `db:"uptime_last_checked" json:"uptime_last_checked" description:"The bot's last uptime check"`
	ClaimedBy           pgtype.Text             `db:"claimed_by" json:"claimed_by" description:"The user who claimed the bot"`
	Note                pgtype.Text             `db:"approval_note" json:"approval_note" description:"The note for the bot's approval"`
	CreatedAt           pgtype.Timestamptz      `db:"created_at" json:"created_at" description:"The bot's creation date"`
	LastClaimed         pgtype.Timestamptz      `db:"last_claimed" json:"last_claimed" description:"The bot's last claimed date"`
	LegacyWebhooks      bool                    `db:"-" json:"legacy_webhooks" description:"Whether the bot is using legacy v1 webhooks or not" ci:"internal"` // Must be parsed internally
	TeamOwnerID         pgtype.UUID             `db:"team_owner" json:"-"`
	TeamOwner           *Team                   `db:"-" json:"team_owner" description:"If the bot is in a team, who owns the bot. If not in a team, this will be null and owner will instead be set" ci:"internal"` // Must be parsed internally
	CaptchaOptOut       bool                    `db:"captcha_opt_out" json:"captcha_opt_out" description:"Whether the bot should have captchas shown if the user has captcha_sponsor_enabled"`
}

// @ci table=bots, unfilled=1
//
// CreateBot represents the data sent for the creation of a bot.
type CreateBot struct {
	BotID      string   `db:"bot_id" json:"bot_id" validate:"required,numeric" msg:"Bot ID must be numeric"`                                       // impld
	ClientID   string   `db:"client_id" json:"client_id" validate:"required,numeric" msg:"Client ID must be numeric"`                              // impld
	Short      string   `db:"short" json:"short" validate:"required,min=30,max=150" msg:"Short description must be between 30 and 150 characters"` // impld
	Long       string   `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                 // impld
	Prefix     string   `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`              // impld
	Invite     string   `db:"invite" json:"invite" validate:"required,https" msg:"Invite is required and must be a valid HTTPS URL"`               // impld
	Banner     *string  `db:"banner" json:"banner" validate:"omitempty,https" msg:"Background must be a valid HTTPS URL"`                          // impld
	Library    string   `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`           // impld
	ExtraLinks []Link   `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                   // Impld
	Tags       []string `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	NSFW       bool     `db:"nsfw" json:"nsfw"`
	StaffNote  *string  `db:"approval_note" json:"staff_note" validate:"omitempty,max=512" msg:"Staff note must be less than 512 characters if sent"` // impld
	TeamOwner  string   `db:"-" json:"team_owner" ci:"internal"`                                                                                      // May or may not be present

	// Not needed to send, resolved in backend
	Owner      string      `db:"owner" json:"-"`
	GuildCount *int        `db:"servers" json:"-"`
	VanityRef  pgtype.UUID `db:"vanity_ref" json:"-"`
}

type BotSettingsUpdate struct {
	Short         string   `db:"short" json:"short" validate:"required,min=30,max=150" msg:"Short description must be between 30 and 150 characters"` // impld
	Long          string   `db:"long" json:"long" validate:"required,min=500" msg:"Long description must be at least 500 characters"`                 // impld
	Prefix        string   `db:"prefix" json:"prefix" validate:"required,min=1,max=10" msg:"Prefix must be between 1 and 10 characters"`              // impld
	Invite        string   `db:"invite" json:"invite" validate:"required,https" msg:"Invite is required and must be a valid HTTPS URL"`               // impld
	Banner        *string  `db:"banner" json:"banner" validate:"omitempty,https" msg:"Background must be a valid HTTPS URL"`                          // impld
	Library       string   `db:"library" json:"library" validate:"required,min=1,max=50" msg:"Library must be between 1 and 50 characters"`           // impld
	ExtraLinks    []Link   `db:"extra_links" json:"extra_links" validate:"required" msg:"Extra links must be sent"`                                   // Impld
	Tags          []string `db:"tags" json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	NSFW          bool     `db:"nsfw" json:"nsfw"`
	CaptchaOptOut bool     `db:"captcha_opt_out" json:"captcha_opt_out"`
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

type DiscordBotMeta struct {
	BotID       string   `json:"bot_id" description:"The bot's ID"`
	ClientID    string   `json:"client_id" description:"The bot's client ID"`
	Name        string   `json:"name" description:"The bot's name"`
	Avatar      string   `json:"avatar" description:"The bot's avatar"`
	ListType    string   `json:"list_type" description:"If this is empty, then it is not on the list"`
	GuildCount  int      `json:"guild_count" description:"The bot's guild count"`
	BotPublic   bool     `json:"bot_public" description:"Whether or not the bot is public"`
	Flags       []string `json:"flags" description:"The bot's flags"`
	Description string   `json:"description" description:"The suggested description for the bot"`
	Tags        []string `json:"tags" description:"The suggested tags for the bot"`
	Fallback    bool     `json:"fallback" description:"Whether or not we had to fallback to RPC from JAPI.rest"`
}

type PatchBotTeam struct {
	TeamID string `json:"team_id" description:"The team ID to add the bot to"`
}
