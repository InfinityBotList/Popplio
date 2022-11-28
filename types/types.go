package types

import (
	"popplio/state"
	"strconv"
	"time"

	"reflect"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgtype"
)

// A link is any extra link
type Link struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Interval struct {
	Hours        float64 `json:"hr"`
	Minutes      float64 `json:"min"`
	Seconds      float64 `json:"sec"`
	Milliseconds int64   `json:"ms"`
	Microseconds int64   `json:"us"`
}

func NewInterval(d time.Duration) Interval {
	return Interval{
		Hours:        d.Hours(),
		Minutes:      d.Minutes(),
		Seconds:      d.Seconds(),
		Milliseconds: d.Milliseconds(),
		Microseconds: d.Microseconds(),
	}
}

// SEO object (minified bot/user/server for seo purposes)
type SEO struct {
	Username string `json:"username"`
	ID       string `json:"id"`
	Avatar   string `json:"avatar"`
	Short    string `json:"short"`
}

type ResolvedPackBot struct {
	User         *DiscordUser `json:"user"`
	Short        string       `json:"short"`
	Type         pgtype.Text  `json:"type"`
	Vanity       pgtype.Text  `json:"vanity"`
	Banner       pgtype.Text  `json:"banner"`
	NSFW         bool         `json:"nsfw"`
	Premium      bool         `json:"premium"`
	Shards       int          `json:"shards"`
	Votes        int          `json:"votes"`
	InviteClicks int          `json:"invites"`
	Servers      int          `json:"servers"`
	Tags         []string     `json:"tags"`
}

type PackVote struct {
	UserID    string    `json:"user_id"`
	Upvote    bool      `json:"upvote"`
	CreatedAt time.Time `json:"created_at"`
}

type BotPack struct {
	Owner         string            `db:"owner" json:"owner_id"`
	ResolvedOwner *DiscordUser      `db:"-" json:"owner"`
	Name          string            `db:"name" json:"name"`
	Short         string            `db:"short" json:"short"`
	Votes         []PackVote        `db:"-" json:"votes"`
	Tags          []string          `db:"tags" json:"tags"`
	URL           string            `db:"url" json:"url"`
	CreatedAt     time.Time         `db:"created_at" json:"created_at"`
	Bots          []string          `db:"bots" json:"bot_ids"`
	ResolvedBots  []ResolvedPackBot `db:"-" json:"bots"`
}

type IndexBotPack struct {
	Owner     string     `db:"owner" json:"owner_id"`
	Name      string     `db:"name" json:"name"`
	Short     string     `db:"short" json:"short"`
	Votes     []PackVote `db:"-" json:"votes"`
	Tags      []string   `db:"tags" json:"tags"`
	URL       string     `db:"url" json:"url"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	Bots      []string   `db:"bots" json:"bot_ids"`
}

type AllPacks struct {
	Count    uint64         `json:"count"`
	PerPage  uint64         `json:"per_page"`
	Next     string         `json:"next"`
	Previous string         `json:"previous"`
	Results  []IndexBotPack `json:"packs"`
}

// A review is a review on ibl
// TODO: Make a review_votes table for holding votes
type Review struct {
	ITag      pgtype.UUID `db:"itag" json:"itag"`
	ID        pgtype.UUID `db:"id" json:"id"`
	BotID     string      `db:"bot_id" json:"bot_id"`
	Author    string      `db:"author" json:"author"`
	Content   string      `db:"content" json:"content"`
	StarRate  pgtype.Int4 `db:"stars" json:"stars"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	Replies   []Reply     `db:"-" json:"replies"`
}

// TODO: Implement replies
type Reply struct {
	ITag      pgtype.UUID `db:"itag" json:"itag"`
	ID        pgtype.UUID `db:"id" json:"id"`
	Author    string      `db:"author" json:"author"`
	Content   string      `db:"content" json:"content"`
	StarRate  pgtype.Int4 `db:"stars" json:"stars"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	Parent    pgtype.UUID `db:"parent" json:"parent"`
}

type ReviewList struct {
	Reviews []Review `json:"reviews"`
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

type ListIndex struct {
	Certified     []IndexBot     `json:"certified"`
	MostViewed    []IndexBot     `json:"most_viewed"`
	Packs         []IndexBotPack `json:"packs"`
	RecentlyAdded []IndexBot     `json:"recently_added"`
	TopVoted      []IndexBot     `json:"top_voted"`
}

type VoteInfo struct {
	Weekend bool `json:"is_weekend"`
}

type UserVote struct {
	Timestamps   []int64 `json:"ts"`
	VoteTime     uint16  `json:"vote_time"`
	HasVoted     bool    `json:"has_voted"`
	LastVoteTime int64   `json:"last_vote_time"`
}

type UserVoteCompat struct {
	HasVoted bool `json:"hasVoted"`
}

type NotifGet struct {
	Endpoint    string           `json:"endpoint"`
	NotifID     string           `json:"notif_id"`
	CreatedAt   time.Time        `json:"created_at"`
	BrowserInfo NotifBrowserInfo `json:"browser_info"`
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

func (s BotStats) GetStats() (servers uint64, shards uint64, users uint64) {
	var serverCount any
	var shardCount any
	var userCount any

	if s.Servers != nil {
		serverCount = *s.Servers
	} else if s.GuildCount != nil {
		serverCount = *s.GuildCount
	} else if s.ServerCount != nil {
		serverCount = *s.ServerCount
	} else if s.Count != nil {
		serverCount = *s.Count
	} else if s.Guilds != nil {
		serverCount = *s.Guilds
	}

	if s.Shards != nil {
		shardCount = *s.Shards
	} else if s.ShardCount != nil {
		shardCount = *s.ShardCount
	}

	if s.Users != nil {
		userCount = *s.Users
	} else if s.UserCount != nil {
		userCount = *s.UserCount
	}

	var serversParsed uint64
	var shardsParsed uint64
	var usersParsed uint64

	// Handle uint64 by converting to uint32
	if serverInt, ok := serverCount.(uint64); ok {
		serversParsed = serverInt
	}

	if shardInt, ok := shardCount.(uint64); ok {
		shardsParsed = shardInt
	}
	if userInt, ok := userCount.(uint64); ok {
		usersParsed = userInt
	}

	// Handle uint32
	if serverInt, ok := serverCount.(uint32); ok {
		serversParsed = uint64(serverInt)
	}
	if shardInt, ok := shardCount.(uint32); ok {
		shardsParsed = uint64(shardInt)
	}
	if userInt, ok := userCount.(uint32); ok {
		usersParsed = uint64(userInt)
	}

	// Handle float64
	if serverFloat, ok := serverCount.(float64); ok {
		if serverFloat < 0 {
			serversParsed = 0
		} else {
			serversParsed = uint64(serverFloat)
		}
	}
	if shardFloat, ok := shardCount.(float64); ok {
		if shardFloat < 0 {
			shardsParsed = 0
		} else {
			shardsParsed = uint64(shardFloat)
		}
	}
	if userFloat, ok := userCount.(float64); ok {
		if userFloat < 0 {
			userFloat = 0
		} else {
			usersParsed = uint64(userFloat)
		}
	}

	// Handle float32
	if serverFloat, ok := serverCount.(float32); ok {
		serversParsed = uint64(serverFloat)
	}
	if shardFloat, ok := shardCount.(float32); ok {
		shardsParsed = uint64(shardFloat)
	}
	if userFloat, ok := userCount.(float32); ok {
		usersParsed = uint64(userFloat)
	}

	// Handle int64
	if serverInt, ok := serverCount.(int64); ok {
		if serverInt < 0 {
			serversParsed = 0
		} else {
			serversParsed = uint64(serverInt)
		}
	}
	if shardInt, ok := shardCount.(int64); ok {
		if shardInt < 0 {
			shardsParsed = 0
		} else {
			shardsParsed = uint64(shardInt)
		}
	}
	if userInt, ok := userCount.(int64); ok {
		if userInt < 0 {
			usersParsed = 0
		} else {
			usersParsed = uint64(userInt)
		}
	}

	// Handle int32
	if serverInt, ok := serverCount.(int32); ok {
		if serverInt < 0 {
			serversParsed = 0
		} else {
			serversParsed = uint64(serverInt)
		}
	}
	if shardInt, ok := shardCount.(int32); ok {
		if shardInt < 0 {
			shardsParsed = 0
		} else {
			shardsParsed = uint64(shardInt)
		}
	}
	if userInt, ok := userCount.(int32); ok {
		if userInt < 0 {
			usersParsed = 0
		} else {
			usersParsed = uint64(userInt)
		}
	}

	// Handle string
	if serverString, ok := serverCount.(string); ok {
		if serverString == "" {
			serversParsed = 0
		} else {
			serversParsed, _ = strconv.ParseUint(serverString, 10, 64)
		}
	}

	if shardString, ok := shardCount.(string); ok {
		if shardString == "" {
			shardsParsed = 0
		} else {
			shardsParsed, _ = strconv.ParseUint(shardString, 10, 64)
		}
	}

	if userString, ok := userCount.(string); ok {
		if userString == "" {
			usersParsed = 0
		} else {
			usersParsed, _ = strconv.ParseUint(userString, 10, 64)
		}
	}

	state.Logger.With(
		"serverCount", serversParsed,
		"shardCount", shardsParsed,
		"userCount", usersParsed,
		"serversType", reflect.TypeOf(serverCount),
		"shardsType", reflect.TypeOf(shardCount),
		"usersType", reflect.TypeOf(userCount),
	).Info("Parsed stats")

	return serversParsed, shardsParsed, usersParsed
}

type WebhookPost struct {
	BotID  string `json:"bot_id"`
	UserID string `json:"user_id"`
	Test   bool   `json:"test"`
	Votes  int    `json:"votes"`

	// Only present on test webhook API or during sends internally
	URL string `json:"url"`

	// Only present on test webhook API
	URL2 string `json:"url2"`

	// Only present on test webhook API
	Token string `json:"token"`

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

// This represents a IBL Popplio API Error
type ApiError struct {
	Context any    `json:"context,omitempty"`
	Message string `json:"message"`
	Error   bool   `json:"error"`
}

type UserID struct {
	UserID string `db:"userID" json:"user_id"`
}

// An announcement
type Announcement struct {
	ITag         pgtype.UUID `db:"itag" json:"itag"`
	Author       string      `db:"user_id" json:"author"`
	ID           string      `db:"id" json:"id"`
	Title        string      `db:"title" json:"title"`
	Content      string      `db:"content" json:"content"`
	LastModified time.Time   `db:"modified_date" json:"last_modified"`
	Status       string      `db:"status" json:"status"`
	Targetted    bool        `db:"targetted" json:"targetted"`
	Target       pgtype.Text `db:"target" json:"target"`
}

type AnnouncementList struct {
	Announcements []Announcement `json:"announcements"`
}

// A discord user
type DiscordUser struct {
	ID             string           `json:"id"`
	Username       string           `json:"username"`
	Discriminator  string           `json:"discriminator"`
	Avatar         string           `json:"avatar"`
	Bot            bool             `json:"bot"`
	Mention        string           `json:"mention"`
	Status         discordgo.Status `json:"status"`
	System         bool             `json:"system"`
	Nickname       string           `json:"nickname"`
	Guild          string           `json:"in_guild"`
	Flags          int              `json:"flags"`
	Tag            string           `json:"tag"`
	IsServerMember bool             `json:"is_member"`
}

type Notification struct {
	NotifID string
	Message []byte
}

type NotifBrowserInfo struct {
	// The OS of the browser
	OS         string
	Browser    string
	BrowserVer string
	Mobile     bool
}

type ResolvedReminderBot struct {
	Name   string `db:"-" json:"name"`
	Avatar string `db:"-" json:"avatar"`
}

type Reminder struct {
	UserID      string              `db:"user_id" json:"user_id"`
	BotID       string              `db:"bot_id" json:"bot_id"`
	ResolvedBot ResolvedReminderBot `db:"-" json:"resolved"`
	CreatedAt   time.Time           `db:"created_at" json:"created_at"`
	LastAcked   time.Time           `db:"last_acked" json:"last_acked"`
}

type Message struct {
	Message string `json:"message"`
	Title   string `json:"title"`
	Icon    string `json:"icon"`
}

type DiscordLog struct {
	Message   *discordgo.MessageSend
	ChannelID string
}

type ProfileUpdate struct {
	About      string `json:"bio"`
	ExtraLinks []Link `json:"extra_links"`
}

type ListStatsBot struct {
	BotID              string   `json:"bot_id"`
	Vanity             string   `json:"vanity"`
	Short              string   `json:"short"`
	Type               string   `json:"type"`
	MainOwnerID        string   `json:"main_owner_id"`
	AdditionalOwnerIDS []string `json:"additional_owners_ids"`
}

type ListStats struct {
	Bots         []ListStatsBot `json:"bots"`
	TotalStaff   int64          `json:"total_staff"`
	TotalUsers   int64          `json:"total_users"`
	TotalVotes   int64          `json:"total_votes"`
	TotalPacks   int64          `json:"total_packs"`
	TotalTickets int64          `json:"total_tickets"`
}

// For documentation
type OpenAPI struct{}

type AuthUser struct {
	Token       string       `json:"token"`
	AccessToken string       `json:"access_token"`
	User        *DiscordUser `json:"user"`
}

type AuthInfo struct {
	ClientID string `json:"client_id"`
}

type Transcript struct {
	ID       int              `json:"id"`
	Data     []map[string]any `json:"data"`
	ClosedBy map[string]any   `json:"closed_by"`
	OpenedBy map[string]any   `json:"opened_by"`
}

type UserSubscription struct {
	Auth     string `json:"auth"`
	P256dh   string `json:"p256dh"`
	Endpoint string `json:"endpoint"`
}

type NotifGetList struct {
	Notifications []NotifGet `json:"notifications"`
}

type ReminderList struct {
	Reminders []Reminder `json:"reminders"`
}

type NotificationInfo struct {
	PublicKey string `json:"public_key"`
}

type TargetType int

const (
	TargetTypeUser TargetType = iota
	TargetTypeBot
	TargetTypeServer
)
