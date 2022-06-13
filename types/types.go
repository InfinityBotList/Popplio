package types

import (
	"strconv"
	"time"

	"reflect"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// A bot is a Discord bot that is on the infinity botlist.
type Bot struct {
	ObjID            primitive.ObjectID `bson:"_id" json:"_id"`
	BotID            string             `bson:"botID" json:"bot_id"`
	Name             string             `bson:"botName" json:"name"`
	Avatar           string             `bson:"avatar,omitempty" json:"avatar"`
	TagsRaw          string             `bson:"tags" json:"-"`
	Tags             []string           `bson:"-" json:"tags"` // This is created by API
	Prefix           *string            `bson:"prefix" json:"prefix"`
	Owner            string             `bson:"main_owner" json:"owner"`
	AdditionalOwners []string           `bson:"additional_owners" json:"additional_owners"`
	StaffBot         bool               `bson:"staff" json:"staff_bot"`
	Short            string             `bson:"short" json:"short"`
	Long             string             `bson:"long" json:"long"`
	Library          *string            `bson:"library" json:"library"`
	Website          *string            `bson:"website" json:"website"`
	Donate           *string            `bson:"donate" json:"donate"`
	Support          *string            `bson:"support" json:"support"`
	NSFW             bool               `bson:"nsfw" json:"nsfw"`
	Premium          bool               `bson:"premium" json:"premium"`
	Certified        bool               `bson:"certified" json:"certified"`
	PendingCert      bool               `bson:"pending_cert" json:"pending_cert"`
	Servers          int                `bson:"servers" json:"servers"`
	Shards           int                `bson:"shards" json:"shards"`
	Users            int                `bson:"users" json:"users"`
	Votes            int                `bson:"votes" json:"votes"`
	Views            int                `bson:"clicks" json:"views"`
	InviteClicks     int                `bson:"invite_clicks" json:"invites"`
	Github           *string            `bson:"github" json:"github"`
	Banner           *string            `bson:"background" json:"banner"`
	Invite           *string            `bson:"invite" json:"invite"`
	Type             string             `bson:"type" json:"type"` // For auditing reasons, we do not filter out denied/banned bots in API
	Vanity           string             `bson:"vanity" json:"vanity"`
	ExternalSource   string             `bson:"external_source,omitempty" json:"external_source"`
	ListSource       string             `bson:"listSource,omitempty" json:"list_source"`
	VoteBanned       bool               `bson:"vote_banned,omitempty" json:"vote_banned"`
	CrossAdd         bool               `bson:"cross_add,omitempty" json:"cross_add"`
	StartPeriod      int                `bson:"start_period,omitempty" json:"start_premium_period"`
	SubPeriod        int                `bson:"sub_period,omitempty" json:"premium_period_length"`
	CertReason       string             `bson:"cert_reason,omitempty" json:"cert_reason"`
	Announce         bool               `bson:"announce,omitempty" json:"announce"`
	AnnounceMessage  string             `bson:"announce_msg,omitempty" json:"announce_message"`
	Uptime           int                `bson:"uptime,omitempty" json:"uptime"`
	TotalUptime      int                `bson:"total_uptime,omitempty" json:"total_uptime"`
	Claimed          bool               `bson:"claimed,omitempty" json:"claimed"`
	ClaimedBy        string             `bson:"claimedBy,omitempty" json:"claimed_by"`
	Note             string             `bson:"note,omitempty" json:"approval_note"`
}

type AllBots struct {
	Count    int64  `json:"count"`
	PerPage  uint64 `json:"per_page"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []Bot  `json:"bots"`
}

// A review is a review on ibl
type Review struct {
	ObjID       primitive.ObjectID `bson:"_id" json:"_id"`
	BotID       string             `bson:"botID" json:"bot_id"`
	Author      string             `bson:"author" json:"author"`
	Content     string             `bson:"content" json:"content"`
	Rate        bool               `bson:"rate,omitempty" json:"rate"`
	StarRate    int                `bson:"star_rate,omitempty" json:"stars"`
	LikesRaw    map[string]any     `bson:"likes,omitempty" json:"likes"`
	DislikesRaw map[string]any     `bson:"dislikes,omitempty" json:"dislikes"`
	Date        int                `bson:"date,omitempty" json:"date"`
	Replies     map[string]any     `bson:"replies,omitempty" json:"replies"`
	Editted     bool               `bson:"editted,omitempty" json:"editted"`
	Flagged     bool               `bson:"flagged,omitempty" json:"flagged"`
}

type User struct {
	ObjID     primitive.ObjectID `bson:"_id" json:"_id"`
	ID        string             `bson:"userID" json:"user_id"`
	Votes     map[string]any     `bson:"votes,omitempty" json:"-"` // Not sent due to privacy reasons
	PackVotes map[string]any     `bson:"pack_votes,omitempty" json:"pack_votes"`
	Staff     bool               `bson:"staff,omitempty" json:"staff"`
	Certified bool               `bson:"certified,omitempty" json:"certified"`
	Developer bool               `bson:"developer,omitempty" json:"developer"`
	About     *string            `bson:"about,omitempty" json:"bio"`
	Github    *string            `bson:"github,omitempty" json:"github"`
	Nickname  *string            `bson:"nickname,omitempty" json:"nickname"`
	Website   *string            `bson:"website,omitempty" json:"website"`

	StaffStats    map[string]int `bson:"staff_stats,omitempty" json:"staff_stats"`
	NewStaffStats map[string]int `bson:"new_staff_stats,omitempty" json:"new_staff_stats"`

	VoteBanned bool `bson:"vote_banned,omitempty" json:"vote_banned"`
	Admin      bool `bson:"admin,omitempty" json:"admin"`
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
		serversParsed = uint64(serverFloat)
	}
	if shardFloat, ok := shardCount.(float64); ok {
		shardsParsed = uint64(shardFloat)
	}
	if userFloat, ok := userCount.(float64); ok {
		usersParsed = uint64(userFloat)
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
	Votes  int    `json:"votes,omitempty"`

	// Only present on test webhook API
	URL string `json:"url,omitempty"`

	// Only present on test webhook API
	Token string `json:"token,omitempty"`

	// Only present on test webhook API
	HMACAuth bool `json:"hmac_auth,omitempty"`
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
	UserID string `bson:"userID" json:"user_id"`
}

// An announcement
type Announcement struct {
	ObjID        primitive.ObjectID `bson:"_id" json:"_id"`
	Author       string             `bson:"userID" json:"author"`
	ID           string             `bson:"announcementID" json:"id"`
	Title        string             `bson:"title" json:"title"`
	Content      string             `bson:"content" json:"content"`
	LastModified time.Time          `bson:"modifiedDate" json:"last_modified"`
	Status       string             `bson:"status" json:"status"`
	Targetted    bool               `bson:"targetted" json:"targetted"`
	Target       string             `bson:"target" json:"target"`
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

type Reminder struct {
	UserID    string `bson:"userID" json:"user_id"`
	BotID     string `bson:"botID" json:"bot_id"`
	CreatedAt int64  `bson:"createdAt" json:"created_at"`
	LastAcked int64  `bson:"lastAcked" json:"last_acked"`
}

type Message struct {
	Message string `json:"message"`
	Title   string `json:"title"`
	Icon    string `json:"icon"`
}
