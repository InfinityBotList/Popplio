package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

type PermissionData struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Desc              string   `json:"desc"`
	SupportedEntities []string `json:"supported_entities"`
}

// Represents a team that is an owner of an entity
//
// This struct does not contain all the data present in a team
type EntityTeamOwner struct {
	ID     string `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Avatar string `db:"avatar" json:"avatar"`
}

type Team struct {
	ID       string       `db:"id" json:"id"`
	Name     string       `db:"name" json:"name"`
	Avatar   string       `db:"avatar" json:"avatar"`
	Members  []TeamMember `db:"-" json:"members"`
	UserBots []IndexBot   `json:"user_bots"` // Bots that are owned by the team
}

type TeamMember struct {
	User      *dovetypes.PlatformUser `json:"user"`
	Flags     []string                `json:"flags"`
	CreatedAt time.Time               `json:"created_at"`
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
	Add    []string `json:"add" description:"Add must be the list of permissions to add"`
	Remove []string `json:"remove" description:"Remove must be the list of permissions to remove"`
}
