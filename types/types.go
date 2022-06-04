package types

// A bot is a Discord bot that is on the infinity botlist.
type Bot struct {
	BotID            string   `bson:"botID" json:"bot_id"`
	Name             string   `bson:"botName" json:"name"`
	TagsRaw          string   `bson:"tags" json:"-"`
	Tags             []string `bson:"-" json:"tags"` // This is created by API
	Prefix           *string  `bson:"prefix" json:"prefix"`
	Owner            string   `bson:"main_owner" json:"owner"`
	AdditionalOwners []string `bson:"additional_owners" json:"additional_owners"`
	StaffBot         bool     `bson:"staff" json:"staff_bot"`
	Short            string   `bson:"short" json:"short"`
	Long             string   `bson:"long" json:"long"`
	Library          *string  `bson:"library" json:"library"`
	Website          *string  `bson:"website" json:"website"`
	Donate           *string  `bson:"donate" json:"donate"`
	Support          *string  `bson:"support" json:"support"`
	NSFW             bool     `bson:"nsfw" json:"nsfw"`
	Premium          bool     `bson:"premium" json:"premium"`
	Certified        bool     `bson:"certified" json:"certified"`
	Servers          int      `bson:"servers" json:"servers"`
	Shards           int      `bson:"shards" json:"shards"`
	Votes            int      `bson:"votes" json:"votes"`
	Views            int      `bson:"clicks" json:"views"`
	InviteClicks     int      `bson:"invite_clicks" json:"invites"`
	Github           *string  `bson:"github" json:"github"`
	Banner           *string  `bson:"background" json:"banner"`
	Invite           *string  `bson:"invite" json:"invite"`
	Type             string   `bson:"type" json:"type"` // For auditing reasons, we do not filter out denied/banned bots in API
	Vanity           string   `bson:"vanity" json:"vanity"`
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
