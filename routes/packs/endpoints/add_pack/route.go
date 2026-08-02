// Package add_pack implements PUT /users/{id}/packs — "Create Pack".
//
// Creates a pack. Returns 204 on success
package add_pack

import (
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"slices"
	"strings"
	"unicode"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(CreatePack{})

type CreatePack struct {
	Name    string   `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	URL     string   `json:"url" validate:"required,min=3,max=20,nospaces,notblank,alpha" msg:"URL must be between 3 and 20 characters without spaces and must be alphabetic"`
	Short   string   `json:"short" validate:"required,min=10,max=100" msg:"Description must be between 10 and 100 characters"`
	Tags    []string `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	Bots    []string `json:"bots" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 bots without duplicates"`
	Servers []string `json:"servers" validate:"omitempty,unique,max=10,dive,numeric" msg:"There can be at most 10 servers without duplicates"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Pack",
		Description: "Creates a pack. Returns 204 on success",
		Req:         CreatePack{},
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload CreatePack

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

	if len(payload.Bots)+len(payload.Servers) == 0 {
		return resp.BadRequest("A pack must contain at least one bot or server")
	}

	// Both columns are NOT NULL — a nil Go slice encodes as SQL NULL, so
	// normalize an omitted field to an empty slice before it ever reaches a query.
	if payload.Bots == nil {
		payload.Bots = []string{}
	}
	if payload.Servers == nil {
		payload.Servers = []string{}
	}

	// Strip out unicode characters and validate pack URL
	payload.URL = strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, payload.URL)

	systems, err := validators.GetWordBlacklistSystems(d.Context, payload.URL)

	if err != nil {
		state.Logger.Error("Error while getting word blacklist systems", zap.Error(err), zap.String("userID", d.Auth.ID))
		return resp.BadRequest("Error while getting word blacklist systems: " + err.Error())
	}

	if slices.Contains(systems, "pack.url") {
		return resp.BadRequest("The chosen pack url is blacklisted")
	}

	// Check that all bots exist
	for _, bot := range payload.Bots {
		botUser, err := dovewing.GetUser(d.Context, bot, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.BadRequest("One of the bot you wish to add does not exist [" + bot + "]: " + err.Error())
		}

		if !botUser.Bot {
			return resp.BadRequest("One of the bot you wish to add is not actually a bot [" + bot + "]")
		}
	}

	// Check that all servers exist. Anyone may add any existing bot/server to
	// a pack — packs are curated lists, not something scoped to what the
	// author owns.
	for _, server := range payload.Servers {
		var count int64

		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE server_id = $1", server).Scan(&count)

		if err != nil {
			return resp.ErrBody("Error checking if server exists:", "Error checking if server exists: "+err.Error(), err)
		}

		if count == 0 {
			return resp.BadRequest("One of the servers you wish to add does not exist [" + server + "]")
		}
	}

	// Check that the pack does not already exist
	var count int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", payload.URL).Scan(&count)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	if count > 0 {
		return resp.BadRequest("A pack with that URL already exists")
	}

	// Create the pack
	_, err = state.Pool.Exec(
		d.Context,
		"INSERT INTO packs (name, url, short, tags, bots, servers, owner) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		payload.Name,
		payload.URL,
		payload.Short,
		payload.Tags,
		payload.Bots,
		payload.Servers,
		d.Auth.ID,
	)

	if err != nil {
		return resp.BadRequest(err.Error())
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
