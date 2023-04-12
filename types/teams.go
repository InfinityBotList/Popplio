package types

import (
	"popplio/teams"
	"time"

	"github.com/infinitybotlist/eureka/dovewing"
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
