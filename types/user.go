package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserExperiment string

const (
	ServerListingUserExperiment UserExperiment = "SERVER_LISTING"
)

// @ci table=users
type User struct {
	ITag                  pgtype.UUID             `db:"itag" json:"itag" description:"The user's internal ID. An artifact of database migrations."`
	ID                    string                  `db:"user_id" json:"-"`
	User                  *dovetypes.PlatformUser `db:"-" json:"user" ci:"internal"` // Must be handled internally
	Experiments           []string                `db:"experiments" json:"experiments" description:"The experiments the user is in"`
	Certified             bool                    `db:"certified" json:"certified"`
	BotDeveloper          bool                    `db:"developer" json:"bot_developer"`
	BugHunters            bool                    `db:"bug_hunters" json:"bug_hunters"`
	CaptchaSponsorEnabled bool                    `db:"captcha_sponsor_enabled" json:"captcha_sponsor_enabled"`
	ExtraLinks            []Link                  `db:"extra_links" json:"extra_links" description:"The users links that it wishes to advertise"`
	About                 pgtype.Text             `db:"about" json:"about"`
	VoteBanned            bool                    `db:"vote_banned" json:"vote_banned"`
	Banned                bool                    `db:"banned" json:"banned"`
	Staff                 bool                    `db:"-" json:"staff" ci:"internal"`                                                   // Must be handled internally
	UserTeams             []Team                  `db:"-" json:"user_teams" ci:"internal"`                                              // Must be handled internally
	UserBots              []IndexBot              `db:"-" json:"user_bots" ci:"internal"`                                               // Must be handled internally
	UserPacks             []BotPack               `db:"-" json:"user_packs" description:"The list of packs the user has" ci:"internal"` // Must be handled internally
	CreatedAt             time.Time               `db:"created_at" json:"created_at" description:"The time the user was created"`
	UpdatedAt             time.Time               `db:"updated_at" json:"updated_at" description:"The time the user was last updated"`
}

type UserPerm struct {
	ID                    string                  `db:"user_id" json:"-"`
	User                  *dovetypes.PlatformUser `db:"-" json:"user"`                // Must be handled internally
	Staff                 bool                    `db:"-" json:"staff" ci:"internal"` // Must be handled internally
	Experiments           []string                `db:"experiments" json:"experiments"`
	Banned                bool                    `db:"banned" json:"banned"`
	CaptchaSponsorEnabled bool                    `db:"captcha_sponsor_enabled" json:"captcha_sponsor_enabled"`
	VoteBanned            bool                    `db:"vote_banned" json:"vote_banned"`
}

// @ci table=staff_members unfilled=1
type StaffMember struct {
	ID            string                  `db:"user_id" json:"-"`
	User          *dovetypes.PlatformUser `db:"-" json:"user" ci:"internal"` // Must be handled internally
	PositionIDs   []pgtype.UUID           `db:"positions" json:"-"`
	Positions     []StaffPosition         `db:"-" json:"positions" ci:"internal"` // Must be handled internally
	PermOverrides []string                `db:"perm_overrides" json:"perm_overrides"`
	NoAutosync    bool                    `db:"no_autosync" json:"no_autosync"`
	MFAVerified   bool                    `db:"mfa_verified" json:"mfa_verified"`
	Unaccounted   bool                    `db:"unaccounted" json:"unaccounted"`
	CreatedAt     time.Time               `db:"created_at" json:"created_at"`
}

// @ci table=staff_positions
type StaffPosition struct {
	ID                 pgtype.UUID `db:"id" json:"id"`
	Name               string      `db:"name" json:"name"`
	RoleID             string      `db:"role_id" json:"role_id"`
	Perms              []string    `db:"perms" json:"perms"`
	CreatedAt          time.Time   `db:"created_at" json:"created_at"`
	Index              int         `db:"index" json:"index"`
	CorrespondingRoles []Link      `db:"corresponding_roles" json:"corresponding_roles"`
	Icon               string      `db:"icon" json:"icon"`
}

type StaffTeam struct {
	Members []StaffMember `json:"members"`
}

type ProfileUpdate struct {
	About                 string `json:"about"`
	ExtraLinks            []Link `json:"extra_links"`
	CaptchaSponsorEnabled *bool  `json:"captcha_sponsor_enabled"`
}

type BoosterStatus struct {
	Remark    string `json:"remark,omitempty" description:"Any issues found when checking booster status"`
	IsBooster bool   `json:"is_booster" description:"Whether the user is a booster"`
}
