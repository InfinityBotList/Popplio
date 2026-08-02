// Package create_team implements POST /teams — "Create Team".
//
// Creates a team. Returns a 201 with the team ID on success.
package create_team

import (
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/teams"
	"popplio/types"
	"popplio/validators"
	"strings"

	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	compiledMessages = uapi.CompileValidationErrors(types.CreateEditTeam{})
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Team",
		Description: "Creates a team. Returns a 201 with the team ID on success.",
		Params:      []docs.Parameter{},
		Req:         types.CreateEditTeam{},
		Resp:        types.CreateTeamResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload types.CreateEditTeam

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	// Validate the payload
	err := state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	var el = []types.Link{}

	if payload.ExtraLinks != nil {
		err = validators.ValidateExtraLinks(*payload.ExtraLinks)

		if err != nil {
			return resp.BadRequest(err.Error())
		}

		el = *payload.ExtraLinks
	}

	var isTeamNsfw = false

	if payload.NSFW != nil {
		isTeamNsfw = *payload.NSFW
	}

	if payload.Tags != nil {
		tagList := *payload.Tags

		for _, tag := range tagList {
			if cases.Lower(language.English).String(tag) == "nsfw" {
				isTeamNsfw = true
			}
		}
	}

	// Create the team
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error starting transaction", err, zap.String("user_id", d.Auth.ID))
	}

	defer tx.Rollback(d.Context)

	// Create vanity
	vanity := strings.ToLower(payload.Name)

	var repl = [][2]string{
		{" ", "-"},
		{"_", "-"},
		{".", ""},
	}

	for _, r := range repl {
		vanity = strings.ReplaceAll(vanity, r[0], r[1])
	}

	var count int64
	err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE code = $1", vanity).Scan(&count)

	if err != nil {
		return resp.Err("Error while checking vanity", err, zap.String("userID", d.Auth.ID), zap.String("vanity", vanity))
	}

	for count > 0 {
		newVanity := vanity + "-" + crypto.RandString(8)

		var nc int64
		err = tx.QueryRow(d.Context, "SELECT COUNT(*) FROM vanity WHERE code = $1", newVanity).Scan(&nc)

		if err != nil {
			return resp.Err("Error while checking vanity", err, zap.String("userID", d.Auth.ID), zap.String("vanity", vanity))
		}

		if nc == 0 {
			vanity = newVanity
			break
		}
	}

	var teamId = uuid.New().String()

	if teamId == "" {
		return resp.Err("Error generating team ID", err, zap.String("user_id", d.Auth.ID))
	}

	var itag pgtype.UUID
	err = tx.QueryRow(d.Context, "INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, $3) RETURNING itag", vanity, teamId, "team").Scan(&itag)

	if err != nil {
		return resp.Err("Error while inserting vanity", err, zap.String("userID", d.Auth.ID), zap.String("teamId", teamId), zap.String("vanity", vanity))
	}

	_, err = tx.Exec(d.Context, "INSERT INTO teams (id, name, short, tags, extra_links, nsfw, vanity_ref) VALUES ($1, $2, $3, $4, $5, $6, $7)", teamId, payload.Name, payload.Short, payload.Tags, el, isTeamNsfw, itag)

	if err != nil {
		return resp.Err("Error creating team", err, zap.String("user_id", d.Auth.ID))
	}

	// Add the user to the team
	_, err = tx.Exec(d.Context, "INSERT INTO team_members (team_id, user_id, flags, data_holder) VALUES ($1, $2, $3, true)", teamId, d.Auth.ID, []string{"global." + teams.PermissionOwner})

	if err != nil {
		return resp.Err("Error adding user to team", err, zap.String("user_id", d.Auth.ID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("user_id", d.Auth.ID))
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: types.CreateTeamResponse{
			TeamID: teamId,
		},
	}
}
