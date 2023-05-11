package types

import (
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserExperiment string

const (
	ServerListingUserExperiment UserExperiment = "SERVER_LISTING"
)

type UserLogin struct {
	Token  string `json:"token" description:"The users token"`
	UserID string `json:"user_id" description:"The users ID"`
}

// @ci table=users
type User struct {
	ITag                             pgtype.UUID           `db:"itag" json:"itag" description:"The user's internal ID. An artifact of database migrations."`
	ID                               string                `db:"user_id" json:"-"`
	User                             *dovewing.DiscordUser `db:"-" json:"user" ci:"internal"` // Must be handled internally
	Experiments                      []string              `db:"experiments" json:"experiments" description:"The experiments the user is in"`
	StaffOnboarded                   bool                  `db:"staff_onboarded" json:"staff_onboarded"`
	StaffOnboardState                string                `db:"staff_onboard_state" json:"staff_onboard_state"`
	StaffOnboardLastStartTime        pgtype.Timestamptz    `db:"staff_onboard_last_start_time" json:"staff_onboard_last_start_time"`
	StaffOnboardGuild                pgtype.Text           `db:"staff_onboard_guild" json:"staff_onboard_guild"`
	StaffRPCLastVerify               pgtype.Timestamptz    `db:"staff_rpc_last_verify" json:"staff_rpc_last_verify"`
	StaffOnboardSessionCode          pgtype.Text           `pdb:"staff_onboard_session_code" json:"-"`
	StaffOnboardCurrentOnboardRespId pgtype.Text           `pdb:"staff_onboard_current_onboard_resp_id" json:"-"`
	Staff                            bool                  `db:"staff" json:"staff"`
	Admin                            bool                  `db:"admin" json:"admin"`
	HAdmin                           bool                  `db:"hadmin" json:"hadmin"`
	Certified                        bool                  `db:"certified" json:"certified"`
	IBLDev                           bool                  `db:"ibldev" json:"ibldev"`
	IBLHDev                          bool                  `db:"iblhdev" json:"iblhdev"`
	Owner                            bool                  `db:"owner" json:"owner"`
	BotDeveloper                     bool                  `db:"developer" json:"bot_developer"`
	BugHunters                       bool                  `db:"bug_hunters" json:"bug_hunters"`
	CaptchaSponsorEnabled            bool                  `db:"captcha_sponsor_enabled" json:"captcha_sponsor_enabled"`
	ExtraLinks                       []Link                `db:"extra_links" json:"extra_links" description:"The users links that it wishes to advertise"`
	About                            pgtype.Text           `db:"about" json:"about"`
	VoteBanned                       bool                  `db:"vote_banned" json:"vote_banned"`
	Banned                           bool                  `db:"banned" json:"banned"`
	UserBots                         []UserBot             `json:"user_bots" ci:"internal"`  // Must be handled internally
	UserTeams                        []Team                `json:"user_teams" ci:"internal"` // Must be handled internally
	UserPacks                        []IndexBotPack        `json:"user_packs" ci:"internal"` // Must be handled internally
}

type UserBot struct {
	BotID        string                `db:"bot_id" json:"bot_id"`
	User         *dovewing.DiscordUser `db:"-" json:"user"`
	Short        string                `db:"short" json:"short"`
	Type         string                `db:"type" json:"type"`
	Vanity       string                `db:"vanity" json:"vanity"`
	Votes        int                   `db:"votes" json:"votes"`
	Shards       int                   `db:"shards" json:"shards"`
	Library      string                `db:"library" json:"library"`
	InviteClicks int                   `db:"invite_clicks" json:"invite_clicks"`
	Views        int                   `db:"clicks" json:"clicks"`
	Servers      int                   `db:"servers" json:"servers"`
	NSFW         bool                  `db:"nsfw" json:"nsfw"`
	Tags         []string              `db:"tags" json:"tags"`
	Premium      bool                  `db:"premium" json:"premium"`
}

type UserPerm struct {
	ID                    string                `db:"user_id" json:"-"`
	User                  *dovewing.DiscordUser `db:"-" json:"user"` // Must be handled internally
	Experiments           []string              `db:"experiments" json:"experiments"`
	Banned                bool                  `db:"banned" json:"banned"`
	CaptchaSponsorEnabled bool                  `db:"captcha_sponsor_enabled" json:"captcha_sponsor_enabled"`
	VoteBanned            bool                  `db:"vote_banned" json:"vote_banned"`
	Staff                 bool                  `db:"staff" json:"staff"`
	Admin                 bool                  `db:"admin" json:"admin"`
	HAdmin                bool                  `db:"hadmin" json:"hadmin"`
	IBLDev                bool                  `db:"ibldev" json:"ibldev"`
	IBLHDev               bool                  `db:"iblhdev" json:"iblhdev"`
	Owner                 bool                  `db:"owner" json:"owner"`
}
