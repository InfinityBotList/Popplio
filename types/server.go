package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// @ci table=servers
//
// Server represents a server.
type Server struct {
	ServerID            string             `db:"server_id" json:"server_id" description:"The server's ID"`
	Name                string             `db:"name" json:"name" description:"The server's name"`
	Avatar              string             `db:"avatar" json:"avatar" description:"The server's avatar"`
	TotalMembers        int                `db:"total_members" json:"total_members" description:"The server's total member count"`
	OnlineMembers       int                `db:"online_members" json:"online_members" description:"The server's online member count"`
	Short               string             `db:"short" json:"short" description:"The server's short description"`
	Long                string             `db:"long" json:"long" description:"The server's long description in raw format (HTML/markdown etc. based on the servers settings)"`
	Type                string             `db:"type" json:"type" description:"The server's type (e.g. pending/approved/certified/denied etc.)"`
	State               string             `db:"state" json:"state" description:"The server's state (public, private, unlisted)"`
	Flags               []string           `db:"flags" json:"flags" description:"The server's flags"`
	VanityRef           pgtype.UUID        `db:"vanity_ref" json:"vanity_ref"`
	Vanity              string             `db:"-" json:"vanity" description:"The server's vanity URL" ci:"internal"` // Must be parsed internally
	ExtraLinks          []Link             `db:"extra_links" json:"extra_links" description:"The server's links that it wishes to advertise"`
	TeamOwnerID         pgtype.UUID        `db:"team_owner" json:"-"`
	TeamOwner           *Team              `db:"-" json:"team_owner" description:"If the server is in a team, who owns the server." ci:"internal"` // Must be parsed internally
	InviteClicks        int                `db:"invite_clicks" json:"invite_clicks" description:"The server's invite click count (via users inviting the server from IBL)"`
	Banner              pgtype.Text        `db:"banner" json:"banner" description:"The server's banner URL if it has one, otherwise null"`
	Clicks              int                `db:"clicks" json:"clicks" description:"The server's total click count"`
	UniqueClicks        int64              `db:"-" json:"unique_clicks" description:"The server's unique click count based on SHA256 hashed IPs" ci:"internal"` // Must be parsed internally
	NSFW                bool               `db:"nsfw" json:"nsfw" description:"Whether the serber is NSFW or not"`
	Votes               int                `db:"votes" json:"votes" description:"The server's vote count"`
	VoteBanned          bool               `db:"vote_banned" json:"vote_banned" description:"Whether the server is vote banned or not"`
	Premium             bool               `db:"premium" json:"premium" description:"Whether the server is a premium server or not"`
	StartPeriod         pgtype.Timestamptz `db:"start_premium_period" json:"start_premium_period"`
	PremiumPeriodLength time.Duration      `db:"premium_period_length" json:"premium_period_length" description:"The period of premium for the server in nanoseconds"`
	CaptchaOptOut       bool               `db:"captcha_opt_out" json:"captcha_opt_out" description:"Whether the server should have captchas shown if the user has captcha_sponsor_enabled"`
}
