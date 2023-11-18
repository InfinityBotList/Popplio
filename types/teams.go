package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type PermissionDataOverride struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type PermissionData struct {
	ID                string                             `json:"id"`
	Name              string                             `json:"name"`
	Desc              string                             `json:"desc"`
	SupportedEntities []string                           `json:"supported_entities"`
	DataOverride      map[string]*PermissionDataOverride `json:"data_override,omitempty"`
}

// @ci table=teams
//
// Team represents a team on Infinity List.
type Team struct {
	ID         string         `db:"id" json:"id" description:"The ID of the team"`
	Name       string         `db:"name" json:"name" description:"The name of the team"`
	Avatar     *AssetMetadata `db:"-" json:"avatar" description:"The avatar of the team"`
	Banner     *AssetMetadata `db:"-" json:"banner" description:"Banner information/metadata"`
	Short      pgtype.Text    `db:"short" json:"short" description:"The teams's short description if it has one, otherwise null"`
	Tags       []string       `db:"tags" json:"tags" description:"The teams's tags if it has any, otherwise null"`
	VoteBanned bool           `db:"vote_banned" json:"vote_banned" description:"Whether the team is banned from voting"`
	Votes      int            `db:"votes" json:"votes" description:"The teams's vote count"`
	ExtraLinks []Link         `db:"extra_links" json:"extra_links" description:"The teams's links that it wishes to advertise"`
	Entities   *TeamEntities  `db:"-" json:"entities" description:"The entities of the team"` // Must be handled internally
	NSFW       bool           `db:"nsfw" json:"nsfw" description:"Whether the team is NSFW (primarily makes NSFW content)"`
}

type TeamBulkFetch struct {
	Teams []Team `json:"teams"`
}

type TeamEntities struct {
	Targets []string      `json:"targets,omitempty" description:"The targets available in the response"`
	Members []TeamMember  `json:"members,omitempty" description:"Members of the team"`
	Bots    []IndexBot    `json:"bots,omitempty" description:"Bots of the team"`
	Servers []IndexServer `json:"servers,omitempty" description:"Servers of the team"`
}

type TeamMember struct {
	ITag        pgtype.UUID             `db:"itag" json:"itag" description:"The ID of the team member"`
	UserID      string                  `db:"user_id" json:"-" description:"The ID of the user"`
	User        *dovetypes.PlatformUser `db:"-" json:"user" description:"A user object representing the user"`
	Flags       []string                `db:"flags" json:"flags" description:"The permissions/flags of the team member"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at" description:"The time the team member was added"`
	Mentionable bool                    `db:"mentionable" json:"mentionable" description:"Whether the user is mentionable (for alerts in bot-logs etc.)"`
	DataHolder  bool                    `db:"data_holder" json:"data_holder" description:"Whether the user is a data holder responsible for all data on the team. That is, should performing mass-scale operations on them affect the team"`
}

type CreateEditTeam struct {
	Name       string    `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Short      *string   `json:"short" validate:"omitempty,max=150" msg:"Short description must be a maximum of 150 characters"` // impld
	Tags       *[]string `json:"tags" validate:"omitempty,unique,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There may a maximum of 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	ExtraLinks *[]Link   `json:"extra_links" description:"The team's links that it wishes to advertise"`
	NSFW       *bool     `json:"nsfw" description:"Whether the team is NSFW (primarily makes NSFW content)"`
}

type CreateTeamResponse struct {
	TeamID pgtype.UUID `json:"team_id"`
}

type PermissionResponse struct {
	Perms []PermissionData `json:"perms"`
}

type AddTeamMember struct {
	UserID string   `json:"user_id" description:"The ID of the user to add to the team"`
	Perms  []string `json:"perms" description:"The initial permissions to give to the user"`
}

type EditTeamMember struct {
	PermUpdate  *PermissionUpdate `json:"perm_update" description:"The permissions to update"`
	Mentionable *bool             `json:"mentionable" description:"Whether the user is mentionable Whether the user is mentionable (for alerts in bot-logs etc.)"`
	DataHolder  *bool             `db:"data_holder" json:"data_holder" description:"Whether the user is a data holder responsible for all data on the team. That is, should performing mass-scale operations on them affect the team"`
}

type PermissionUpdate struct {
	Add    []string `json:"add" description:"Add must be the list of permissions to add"`
	Remove []string `json:"remove" description:"Remove must be the list of permissions to remove"`
}

type UserEntityPerms struct {
	Perms []string `json:"perms" description:"The user's permissions on an entity"`
}
