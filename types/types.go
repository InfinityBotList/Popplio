package types

import "go.mongodb.org/mongo-driver/bson/primitive"

// A bot is a Discord bot that is on the infinity botlist.
type Bot struct {
	ObjID            primitive.ObjectID `bson:"_id" json:"_id"`
	BotID            string             `bson:"botID" json:"bot_id"`
	Name             string             `bson:"botName" json:"name"`
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

type BotStats struct {
	// Fields are ordered in the way they are searched
	// The simple servers, shards way
	Servers *uint32 `json:"servers"`
	Shards  *uint32 `json:"shards"`

	// Fates List uses this (server count)
	GuildCount *uint32 `json:"guild_count"`

	// Top.gg uses this (server count)
	ServerCount *uint32 `json:"server_count"`

	// Top.gg and Fates List uses this (shard count)
	ShardCount *uint32 `json:"shard_count"`

	// Rovel Discord List and dlist.gg (kinda) uses this (server count)
	Count *uint32 `json:"count"`

	// Discordbotlist.com uses this (server count)
	Guilds *uint32 `json:"guilds"`

	Users     *uint32 `json:"users"`
	UserCount *uint32 `json:"user_count"`
}

func (s BotStats) GetStats() (servers uint32, shards uint32, users uint32) {
	var serverCount uint32
	var shardCount uint32
	var userCount uint32

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

	return serverCount, shardCount, userCount
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
}
