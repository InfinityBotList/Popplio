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
