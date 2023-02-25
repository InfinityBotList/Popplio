package types

import "github.com/jackc/pgx/v5/pgtype"

type User struct {
	ITag                      pgtype.UUID        `db:"itag" json:"itag"`
	ID                        string             `db:"user_id" json:"-"`
	User                      *DiscordUser       `db:"-" json:"user"` // Must be handled internally
	Experiments               []string           `db:"experiments" json:"experiments"`
	StaffOnboarded            bool               `db:"staff_onboarded" json:"staff_onboarded"`
	StaffOnboardState         string             `db:"staff_onboard_state" json:"staff_onboard_state"`
	StaffOnboardLastStartTime pgtype.Timestamptz `db:"staff_onboard_last_start_time" json:"staff_onboard_last_start_time"`
	StaffOnboardMacroTime     pgtype.Timestamptz `db:"staff_onboard_macro_time" json:"staff_onboard_macro_time"`
	StaffOnboardGuild         pgtype.Text        `db:"staff_onboard_guild" json:"staff_onboard_guild"`
	StaffRPCLastVerify        pgtype.Timestamptz `db:"staff_rpc_last_verify" json:"staff_rpc_last_verify"`
	Staff                     bool               `db:"staff" json:"staff"`
	Admin                     bool               `db:"admin" json:"admin"`
	HAdmin                    bool               `db:"hadmin" json:"hadmin"`
	Certified                 bool               `db:"certified" json:"certified"`
	IBLDev                    bool               `db:"ibldev" json:"ibldev"`
	IBLHDev                   bool               `db:"iblhdev" json:"iblhdev"`
	Owner                     bool               `db:"owner" json:"owner"`
	BotDeveloper              bool               `db:"developer" json:"bot_developer"`
	BugHunters                bool               `db:"bug_hunters" json:"bug_hunters"`
	CaptchaSponsorEnabled     bool               `db:"captcha_sponsor_enabled" json:"captcha_sponsor_enabled"`
	ExtraLinks                []Link             `db:"extra_links" json:"extra_links"`
	About                     pgtype.Text        `db:"about" json:"about"`
	VoteBanned                bool               `db:"vote_banned" json:"vote_banned"`
	Banned                    bool               `db:"banned" json:"banned"`
	UserBots                  []UserBot          `json:"user_bots"`  // Must be handled internally
	UserTeams                 []UserTeam         `json:"user_teams"` // Must be handled internally
	UserPacks                 []IndexBotPack     `json:"user_packs"` // Must be handled internally
}

type UserTeam struct {
	ID     string `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Avatar string `db:"avatar" json:"avatar"`
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
	ID                    string       `db:"user_id" json:"-"`
	User                  *DiscordUser `db:"-" json:"user"` // Must be handled internally
	Experiments           []string     `db:"experiments" json:"experiments"`
	Banned                bool         `db:"banned" json:"banned"`
	CaptchaSponsorEnabled bool         `db:"captcha_sponsor_enabled" json:"captcha_sponsor_enabled"`
	VoteBanned            bool         `db:"vote_banned" json:"vote_banned"`
	Staff                 bool         `db:"staff" json:"staff"`
	Admin                 bool         `db:"admin" json:"admin"`
	HAdmin                bool         `db:"hadmin" json:"hadmin"`
	IBLDev                bool         `db:"ibldev" json:"ibldev"`
	IBLHDev               bool         `db:"iblhdev" json:"iblhdev"`
	Owner                 bool         `db:"owner" json:"owner"`
}
