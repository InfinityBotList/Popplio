package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type EditTeam struct {
	Name    string `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar  string `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
	Mention string `json:"mention" validate:"required" msg:"The user to mention" description:"ID of the user to mention, if wanted"`
}

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

type PartialTeam struct {
	ID     string `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Avatar string `db:"avatar" json:"avatar"`
}

type Team struct {
	ID      string       `db:"id" json:"id" description:"The ID of the team"`
	Name    string       `db:"name" json:"name" description:"The name of the team"`
	Avatar  string       `db:"avatar" json:"avatar" description:"The avatar of the team"`
	Members []TeamMember `db:"-" json:"members" description:"Members of the team"`
	Bots    []IndexBot   `db:"-" json:"bots" ci:"internal"` // Must be handled internally
}

type TeamMember struct {
	ITag        pgtype.UUID             `db:"itag" json:"itag"`
	UserID      string                  `db:"user_id" json:"-"`
	User        *dovetypes.PlatformUser `db:"-" json:"user"`
	Flags       []string                `db:"flags" json:"flags"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at"`
	Mentionable bool                    `db:"mentionable" json:"mentionable"`
}

type CreateTeam struct {
	Name   string `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar string `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
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
	Mentionable *bool             `json:"mentionable" description:"Whether the user is mentionable"`
}

type PermissionUpdate struct {
	Add    []string `json:"add" description:"Add must be the list of permissions to add"`
	Remove []string `json:"remove" description:"Remove must be the list of permissions to remove"`
}
