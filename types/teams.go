package types

import (
	"time"

	"popplio/perms"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

// PermissionData describes one permission that can be granted. It is the public
// shape of a [perms.Definition], and is used for both the team permissions a
// member can hold and the staff permissions a staff role can carry.
type PermissionData struct {
	ID        string `json:"id" description:"The permission's name, as stored in a member's flags"`
	Name      string `json:"name" description:"Human readable label"`
	Desc      string `json:"desc" description:"What holding the permission lets a member do"`
	Category  string `json:"category" description:"The group the permission belongs to, for display"`
	Dangerous bool   `json:"dangerous" description:"Whether the permission hands over broad or irreversible power"`
}

// @ci table=teams
//
// Team represents a team on Omniplex.
type Team struct {
	ID               string        `db:"id" json:"id" description:"The ID of the team"`
	Name             string        `db:"name" json:"name" description:"The name of the team"`
	Short            pgtype.Text   `db:"short" json:"short" description:"The teams's short description if it has one, otherwise null"`
	Tags             []string      `db:"tags" json:"tags" description:"The teams's tags if it has any, otherwise null"`
	VoteBanned       bool          `db:"vote_banned" json:"vote_banned" description:"Whether the team is banned from voting"`
	ApproximateVotes int           `db:"approximate_votes" json:"approximate_votes" description:"The team's approximate vote count."`
	Votes            int           `db:"-" json:"votes" description:"The team's vote count" ci:"internal"` // Votes are retrieved from entity_votes
	ExtraLinks       []Link        `db:"extra_links" json:"extra_links" description:"The teams's links that it wishes to advertise"`
	Entities         *TeamEntities `db:"-" json:"entities" description:"The entities of the team" ci:"internal"` // Must be handled internally
	NSFW             bool          `db:"nsfw" json:"nsfw" description:"Whether the team is NSFW (primarily makes NSFW content)"`
	VanityRef        pgtype.UUID   `db:"vanity_ref" json:"vanity_ref" description:"The corresponding vanities itag, this also works to ensure that all teams have an associated vanity"`
	Vanity           string        `db:"-" json:"vanity" description:"The team's vanity URL" ci:"internal"` // Must be parsed internally
	Service          string        `db:"service" json:"service" description:"The service which added the team (api/infernoplex) etc."`
	CreatedAt        time.Time     `db:"created_at" json:"created_at" description:"The time the team was created"`
	UpdatedAt        time.Time     `db:"updated_at" json:"updated_at" description:"The time the team was last updated"`
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

// @ci table=team_members
//
// Team Member represents a member of a team on Omniplex.
type TeamMember struct {
	ITag        pgtype.UUID             `db:"itag" json:"itag" description:"The ID of the team member"`
	TeamID      string                  `db:"team_id" json:"team_id" description:"The ID of the team"`
	UserID      string                  `db:"user_id" json:"-" description:"The ID of the user"`
	User        *dovetypes.PlatformUser `db:"-" json:"user" description:"A user object representing the user" ci:"internal"` // Must be handled internally
	Flags       []string                `db:"flags" json:"flags" description:"The permissions/flags of the team member"`
	Service     string                  `db:"service" json:"service" description:"The service which added a team member (api/infernoplex) etc."`
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
	TeamID string `json:"team_id" description:"The ID of the created team"`
}

type PermissionResponse struct {
	Perms []PermissionData `json:"perms"`
}

// PermissionDataFrom renders a catalogue for the API, keeping the catalogue's
// own order so that clients can present the permissions as declared.
func PermissionDataFrom(defs []perms.Definition) []PermissionData {
	out := make([]PermissionData, 0, len(defs))

	for _, d := range defs {
		out = append(out, PermissionData{
			ID:        string(d.ID),
			Name:      d.Name,
			Desc:      d.Description,
			Category:  d.Category,
			Dangerous: d.Dangerous,
		})
	}

	return out
}

type AddTeamMember struct {
	UserID string   `json:"user_id" description:"The ID of the user to add to the team"`
	Perms  []string `json:"perms" description:"The initial permissions to give to the user"`
}

type EditTeamMember struct {
	Perms       *[]string `json:"perms" description:"The permissions to set. If empty, will not update"`
	Mentionable *bool     `json:"mentionable" description:"Whether the user is mentionable Whether the user is mentionable (for alerts in bot-logs etc.)"`
	DataHolder  *bool     `db:"data_holder" json:"data_holder" description:"Whether the user is a data holder responsible for all data on the team. That is, should performing mass-scale operations on them affect the team"`
}

type UserEntityPerms struct {
	Perms []string `json:"perms" description:"The user's permissions on an entity"`
}
