package types

import "github.com/jackc/pgx/v5/pgtype"

type User struct {
	ITag                      pgtype.UUID        `db:"itag" json:"itag"`
	ID                        string             `db:"user_id" json:"user_id"`
	User                      *DiscordUser       `db:"-" json:"user"` // Must be handled internally
	Experiments               []string           `db:"experiments" json:"experiments"`
	StaffOnboarded            bool               `db:"staff_onboarded" json:"staff_onboarded"`
	StaffOnboardState         string             `db:"staff_onboard_state" json:"staff_onboard_state"`
	StaffOnboardLastStartTime pgtype.Timestamptz `db:"staff_onboard_last_start_time" json:"staff_onboard_last_start_time"`
	StaffOnboardMacroTime     pgtype.Timestamptz `db:"staff_onboard_macro_time" json:"staff_onboard_macro_time"`
	StaffOnboardGuild         pgtype.Text        `db:"staff_onboard_guild" json:"staff_onboard_guild"`
	Staff                     bool               `db:"staff" json:"staff"`
	Admin                     bool               `db:"admin" json:"admin"`
	HAdmin                    bool               `db:"hadmin" json:"hadmin"`
	Certified                 bool               `db:"certified" json:"certified"`
	Dev                       bool               `db:"ibldev" json:"ibldev"`
	HDev                      bool               `db:"iblhdev" json:"iblhdev"`
	Developer                 bool               `db:"developer" json:"developer"`
	ExtraLinks                []Link             `db:"extra_links" json:"extra_links"`
	About                     pgtype.Text        `db:"about" json:"about"`
	VoteBanned                bool               `db:"vote_banned" json:"vote_banned"`
	Banned                    bool               `db:"banned" json:"banned"`
	UserBots                  []UserBot          `json:"user_bots"`  // Must be handled internally
	UserPacks                 []IndexBotPack     `json:"user_packs"` // Must be handled internally
}

type UserBot struct {
	BotID              string       `db:"bot_id" json:"bot_id"`
	User               *DiscordUser `db:"-" json:"user"`
	Short              string       `db:"short" json:"short"`
	Type               string       `db:"type" json:"type"`
	Vanity             string       `db:"vanity" json:"vanity"`
	Votes              int          `db:"votes" json:"votes"`
	Shards             int          `db:"shards" json:"shards"`
	Library            string       `db:"library" json:"library"`
	InviteClicks       int          `db:"invite_clicks" json:"invite_clicks"`
	Views              int          `db:"clicks" json:"clicks"`
	Servers            int          `db:"servers" json:"servers"`
	NSFW               bool         `db:"nsfw" json:"nsfw"`
	Tags               []string     `db:"tags" json:"tags"`
	OwnerID            string       `db:"owner" json:"owner_id"`
	Premium            bool         `db:"premium" json:"premium"`
	AdditionalOwnerIDS []string     `db:"additional_owners" json:"additional_owner_ids"`
}

type UserPerm struct {
	ID          string       `json:"user_id"`
	User        *DiscordUser `json:"user"` // Must be handled internally
	Experiments []string     `json:"experiments"`
	Staff       bool         `json:"staff"`
	Admin       bool         `json:"admin"`
	HAdmin      bool         `json:"hadmin"`
	IBLDev      bool         `json:"ibldev"`
	IBLHDev     bool         `json:"iblhdev"`
}
