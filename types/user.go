package types

import "github.com/jackc/pgx/v5/pgtype"

type User struct {
	ITag                      pgtype.UUID        `db:"itag" json:"itag"`
	ID                        string             `db:"user_id" json:"user_id"`
	User                      *DiscordUser       `db:"-" json:"user"` // Must be handled internally
	Staff                     bool               `db:"staff" json:"staff"`
	About                     pgtype.Text        `db:"about" json:"about"`
	Experiments               []string           `db:"experiments" json:"experiments"`
	VoteBanned                bool               `db:"vote_banned" json:"vote_banned"`
	Admin                     bool               `db:"admin" json:"admin"`
	HAdmin                    bool               `db:"hadmin" json:"hadmin"`
	Dev                       bool               `db:"ibldev" json:"ibldev"`
	HDev                      bool               `db:"iblhdev" json:"iblhdev"`
	StaffOnboarded            bool               `db:"staff_onboarded" json:"staff_onboarded"`
	StaffOnboardState         string             `db:"staff_onboard_state" json:"staff_onboard_state"`
	StaffOnboardLastStartTime pgtype.Timestamptz `db:"staff_onboard_last_start_time" json:"staff_onboard_last_start_time"`
	StaffOnboardMacroTime     pgtype.Timestamptz `db:"staff_onboard_macro_time" json:"staff_onboard_macro_time"`
	StaffOnboardGuild         pgtype.Text        `db:"staff_onboard_guild" json:"staff_onboard_guild"`
	Certified                 bool               `db:"certified" json:"certified"`
	Developer                 bool               `db:"developer" json:"developer"`
	UserBots                  []UserBot          `json:"user_bots"` // Must be handled internally
	ExtraLinks                []Link             `db:"extra_links" json:"extra_links"`
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
	InviteClick        int          `db:"invite_clicks" json:"invite_clicks"`
	Views              int          `db:"clicks" json:"clicks"`
	Servers            int          `db:"servers" json:"servers"`
	NSFW               bool         `db:"nsfw" json:"nsfw"`
	Tags               []string     `db:"tags" json:"tags"`
	OwnerID            string       `db:"owner" json:"owner_id"`
	Premium            bool         `db:"premium" json:"premium"`
	AdditionalOwnerIDS []string     `db:"additional_owners" json:"additional_owner_ids"`
}