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

type Team struct {
	ID         string        `db:"id" json:"id" description:"The ID of the team"`
	Name       string        `db:"name" json:"name" description:"The name of the team"`
	Avatar     string        `db:"avatar" json:"avatar" description:"The avatar of the team"`
	Banner     pgtype.Text   `db:"banner" json:"banner" description:"The team's banner URL if it has one, otherwise null"`
	Short      pgtype.Text   `db:"short" json:"short" description:"The teams's short description if it has one, otherwise null"`
	Tags       []string      `db:"tags" json:"tags" description:"The teams's tags if it has any, otherwise null"`
	Votes      int           `db:"votes" json:"votes" description:"The teams's vote count"`
	ExtraLinks []Link        `db:"extra_links" json:"extra_links" description:"The teams's links that it wishes to advertise"`
	Entities   *TeamEntities `db:"-" json:"entities" description:"The entities of the team"` // Must be handled internally
}

type TeamEntities struct {
	Targets []string     `json:"targets,omitempty" description:"The targets available in the response"`
	Members []TeamMember `json:"members,omitempty" description:"Members of the team"`
	Bots    []IndexBot   `json:"bots,omitempty" description:"Bots of the team"` // Must be handled internally
}

type TeamMember struct {
	ITag        pgtype.UUID             `db:"itag" json:"itag"`
	UserID      string                  `db:"user_id" json:"-"`
	User        *dovetypes.PlatformUser `db:"-" json:"user"`
	Flags       []string                `db:"flags" json:"flags"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at"`
	Mentionable bool                    `db:"mentionable" json:"mentionable"`
}

type CreateEditTeam struct {
	Name       string    `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar     string    `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
	Banner     *string   `json:"banner" validate:"omitempty,https" msg:"Background must be a valid HTTPS URL"`                   // impld
	Short      *string   `json:"short" validate:"omitempty,max=150" msg:"Short description must be a maximum of 150 characters"` // impld
	Tags       *[]string `json:"tags" validate:"omitempty,unique,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There may a maximum of 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	ExtraLinks *[]Link   `json:"extra_links" description:"The team's links that it wishes to advertise"`
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
