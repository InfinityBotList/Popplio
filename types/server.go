package types

import "github.com/jackc/pgx/v5/pgtype"

type Server struct {
	ServerID      string      `db:"server_id" json:"server_id" description:"The server's ID"`
	Name          string      `db:"name" json:"name" description:"The server's name"`
	Avatar        string      `db:"avatar" json:"avatar" description:"The server's avatar"`
	TotalMembers  int         `db:"total_members" json:"total_members" description:"The server's total member count"`
	OnlineMembers int         `db:"online_members" json:"online_members" description:"The server's online member count"`
	InviteURL     string      `db:"invite_url" json:"-" description:"The server's invite URL"` // Not filled in, as its usually protected by the invite API
	Short         string      `db:"short" json:"short" description:"The server's short description"`
	Long          string      `db:"long" json:"long" description:"The server's long description in raw format (HTML/markdown etc. based on the bots settings)"`
	State         string      `db:"state" json:"state" description:"The server's state (public, private, unlisted)"`
	Vanity        string      `db:"vanity" json:"vanity" description:"The server's vanity URL"`
	ExtraLinks    []Link      `db:"extra_links" json:"extra_links" description:"The server's links that it wishes to advertise"`
	TeamOwnerID   pgtype.UUID `db:"team_owner" json:"-"`
	TeamOwner     *Team       `json:"team_owner" description:"If the server is in a team, who owns the server."` // Must be parsed internally
}
