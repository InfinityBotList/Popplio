package types

import (
	"popplio/teams"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
)

type Team struct {
	ID       string       `db:"id" json:"id"`
	Name     string       `db:"name" json:"name"`
	Avatar   string       `db:"avatar" json:"avatar"`
	Members  []TeamMember `db:"-" json:"members"`
	UserBots []UserBot    `json:"user_bots"` // Bots that are owned by the team
}

type TeamMember struct {
	User      *dovewing.DiscordUser  `json:"user"`
	Perms     []teams.TeamPermission `json:"perms"`
	CreatedAt time.Time              `json:"created_at"`
}

type CreateTeam struct {
	Name   string `json:"name" validate:"required,nonvulgar,min=3,max=32" msg:"Team name must be between 3 and 32 characters long"`
	Avatar string `json:"avatar" validate:"required,https" msg:"Avatar must be a valid HTTPS URL"`
}

type CreateTeamResponse struct {
	TeamID pgtype.UUID `json:"team_id"`
}

type PermissionResponse struct {
	Perms []teams.PermDetailMap `json:"perms"`
}
