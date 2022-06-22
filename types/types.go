package types

import (
	"strconv"
	"time"

	"reflect"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgtype"
	log "github.com/sirupsen/logrus"
)

// A bot is a Discord bot that is on the infinity botlist.
type Bot struct {
	ITag             pgtype.UUID      `db:"itag" json:"itag"`
	BotID            string           `db:"bot_id" json:"bot_id"`
	Name             string           `db:"name" json:"name"`
	Avatar           pgtype.Text      `db:"-" json:"avatar"`
	Tags             pgtype.TextArray `db:"tags" json:"tags"`
	Prefix           pgtype.Text      `db:"prefix" json:"prefix"`
	Owner            string           `db:"owner" json:"owner"`
	AdditionalOwners []string         `db:"additional_owners" json:"additional_owners"`
	StaffBot         bool             `db:"staff" json:"staff_bot"`
	Short            string           `db:"short" json:"short"`
	Long             string           `db:"long" json:"long"`
	Library          pgtype.Text      `db:"library" json:"library"`
	Website          pgtype.Text      `db:"website" json:"website"`
	Donate           pgtype.Text      `db:"donate" json:"donate"`
	Support          pgtype.Text      `db:"support" json:"support"`
	NSFW             bool             `db:"nsfw" json:"nsfw"`
	Premium          bool             `db:"premium" json:"premium"`
	Certified        bool             `db:"certified" json:"certified"`
	PendingCert      bool             `db:"pending_cert" json:"pending_cert"`
	Servers          int              `db:"servers" json:"servers"`
	Shards           int              `db:"shards" json:"shards"`
	Users            int              `db:"users" json:"users"`
	Votes            int              `db:"votes" json:"votes"`
	Views            int              `db:"clicks" json:"views"`
	InviteClicks     int              `db:"invite_clicks" json:"invites"`
	Github           pgtype.Text      `db:"github" json:"github"`
	Banner           pgtype.Text      `db:"background" json:"banner"`
	Invite           pgtype.Text      `db:"invite" json:"invite"`
	Type             string           `db:"type" json:"type"` // For auditing reasons, we do not filter out denied/banned bots in API
	Vanity           pgtype.Text      `db:"vanity" json:"vanity"`
	ExternalSource   pgtype.Text      `db:"external_source" json:"external_source"`
	ListSource       pgtype.Text      `db:"listSource" json:"list_source"`
	VoteBanned       bool             `db:"vote_banned" json:"vote_banned"`
	CrossAdd         bool             `db:"cross_add" json:"cross_add"`
	StartPeriod      int              `db:"start_period" json:"start_premium_period"`
	SubPeriod        int              `db:"sub_period" json:"premium_period_length"`
	CertReason       string           `db:"cert_reason" json:"cert_reason"`
	Announce         pgtype.Text      `db:"announce" json:"announce"`
	AnnounceMessage  string           `db:"announce_msg" json:"announce_message"`
	Uptime           int              `db:"uptime" json:"uptime"`
	TotalUptime      int              `db:"total_uptime" json:"total_uptime"`
	Claimed          bool             `db:"claimed" json:"claimed"`
	ClaimedBy        string           `db:"claimedBy" json:"claimed_by"`
	Note             string           `db:"note" json:"approval_note"`
	Date             pgtype.Date      `db:"date" json:"date"`
}

type AllBots struct {
	Count    uint64 `json:"count"`
	PerPage  uint64 `json:"per_page"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []*Bot `json:"bots"`
}

// A review is a review on ibl
type Review struct {
	ITag        pgtype.UUID    `db:"itag" json:"itag"`
	BotID       string         `db:"botID" json:"bot_id"`
	Author      string         `db:"author" json:"author"`
	Content     string         `db:"content" json:"content"`
	Rate        bool           `db:"rate" json:"rate"`
	StarRate    int            `db:"star_rate" json:"stars"`
	LikesRaw    map[string]any `db:"likes" json:"likes"`
	DislikesRaw map[string]any `db:"dislikes" json:"dislikes"`
	Date        int            `db:"date" json:"date"`
	Replies     map[string]any `db:"replies" json:"replies"`
	Editted     bool           `db:"editted" json:"editted"`
	Flagged     bool           `db:"flagged" json:"flagged"`
}

type User struct {
	ITag      pgtype.UUID    `db:"itag" json:"itag"`
	ID        string         `db:"userID" json:"user_id"`
	Votes     map[string]any `db:"votes" json:"-"` // Not sent due to privacy reasons
	PackVotes map[string]any `db:"pack_votes" json:"pack_votes"`
	Staff     bool           `db:"staff" json:"staff"`
	Certified bool           `db:"certified" json:"certified"`
	Developer bool           `db:"developer" json:"developer"`
	About     pgtype.Text    `db:"about" json:"bio"`
	Github    pgtype.Text    `db:"github" json:"github"`
	Nickname  pgtype.Text    `db:"nickname" json:"nickname"`
	Website   pgtype.Text    `db:"website" json:"website"`

	StaffStats    map[string]int `db:"staff_stats" json:"staff_stats"`
	NewStaffStats map[string]int `db:"new_staff_stats" json:"new_staff_stats"`

	VoteBanned bool `db:"vote_banned" json:"vote_banned"`
	Admin      bool `db:"admin" json:"admin"`
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

// For documentation purposes
type BotStatsTyped struct {
	// Fields are ordered in the way they are searched
	// The simple servers, shards way
	Servers *uint `json:"servers"`
	Shards  *uint `json:"shards"`

	// Fates List uses this (server count)
	GuildCount *uint `json:"guild_count"`

	// Top.gg uses this (server count)
	ServerCount *uint `json:"server_count"`

	// Top.gg and Fates List uses this (shard count)
	ShardCount *uint `json:"shard_count"`

	// Rovel Discord List and dlist.gg (kinda) uses this (server count)
	Count *uint `json:"count"`

	// Discordbotlist.com uses this (server count)
	Guilds *uint `json:"guilds"`

	Users     *uint `json:"users"`
	UserCount *uint `json:"user_count"`
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

	log.Info(reflect.TypeOf(serverCount))

	log.WithFields(log.Fields{
		"servers":     serversParsed,
		"shards":      shardsParsed,
		"users":       usersParsed,
		"serversType": reflect.TypeOf(serverCount),
		"shardsType":  reflect.TypeOf(shardCount),
		"usersType":   reflect.TypeOf(userCount),
	}).Info("Setting stats")

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
	Votes        int    `json:"votes"`
	UserID       string `json:"user"`
	BotID        string `json:"bot"`
	UserIDLegacy string `json:"userID"`
	BotIDLegacy  string `json:"botID"`
	Test         bool   `json:"test"`
	Time         int64  `json:"time"`
}

// This represents a IBL Popplio API Error
type ApiError struct {
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
	Target       string      `db:"target" json:"target"`
}

// A discord user
type DiscordUser struct {
	ID            string           `json:"id"`
	Username      string           `json:"username"`
	Discriminator string           `json:"discriminator"`
	Avatar        string           `json:"avatar"`
	Bot           bool             `json:"bot"`
	Mention       string           `json:"mention"`
	Status        discordgo.Status `json:"status"`
	System        bool             `json:"system"`
	Nickname      string           `json:"nickname"`
	Guild         string           `json:"in_guild"`
	Flags         int              `json:"flags"`
	Tag           string           `json:"tag"`
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
	Name   string `db:"botName" json:"name"`
	Avatar string `db:"avatar" json:"avatar"`
}

type Reminder struct {
	UserID      string              `db:"userID" json:"user_id"`
	BotID       string              `db:"botID" json:"bot_id"`
	ResolvedBot ResolvedReminderBot `db:"-" json:"resolved"`
	CreatedAt   int64               `db:"createdAt" json:"created_at"`
	LastAcked   int64               `db:"lastAcked" json:"last_acked"`
}

type Message struct {
	Message string `json:"message"`
	Title   string `json:"title"`
	Icon    string `json:"icon"`
}

type DiscordLog struct {
	Message     *discordgo.MessageSend
	WebhookData *discordgo.WebhookParams
	ChannelID   string

	// Only for webhooks
	WebhookID    string
	WebhookToken string
}

type ProfileUpdate struct {
	About string `json:"bio"`
}
