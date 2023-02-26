package types

import "time"

type Team struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Avatar   string       `json:"avatar"`
	Members  []TeamMember `json:"members"`
	UserBots []UserBot    `json:"user_bots"` // Bots that are owned by the team
}

type TeamMember struct {
	User      *DiscordUser `json:"user"`
	Perms     []string     `json:"perms"`
	CreatedAt time.Time    `json:"created_at"`
}
