package add_pack

import (
	"net/http"
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
	Name  string   `json:"name" validate:"required,min=3,max=20" msg:"Name must be between 3 and 20 characters"`
	URL   string   `json:"url" validate:"required,min=3,max=20,nospaces,notblank,alpha" msg:"URL must be between 3 and 20 characters without spaces and must be alphabetic"`
	Short string   `json:"short" validate:"required,min=10,max=100" msg:"Description must be between 10 and 100 characters"`
	Tags  []string `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	Bots  []string `json:"bots" validate:"required,unique,min=1,max=10,dive,numeric" msg:"There must be between 1 and 10 bots without duplicates"`
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
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "Error while getting word blacklist systems: " + err.Error()},
		}
	}

	if slices.Contains(systems, "pack.url") {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "The chosen pack url is blacklisted"},
		}
	}

	// Check that all bots exist
	for _, bot := range payload.Bots {
		botUser, err := dovewing.GetUser(d.Context, bot, state.DovewingPlatformDiscord)

		if err != nil {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "One of the bot you wish to add does not exist [" + bot + "]: " + err.Error()},
			}
		}

		if !botUser.Bot {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "One of the bot you wish to add is not actually a bot [" + bot + "]"},
			}
		}
	}

	// Check that the pack does not already exist
	var count int64
	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE url = $1", payload.URL).Scan(&count)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	if count > 0 {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: "A pack with that URL already exists"},
		}
	}

	// Create the pack
	_, err = state.Pool.Exec(
		d.Context,
		"INSERT INTO packs (name, url, short, tags, bots, owner) VALUES ($1, $2, $3, $4, $5, $6)",
		payload.Name,
		payload.URL,
		payload.Short,
		payload.Tags,
		payload.Bots,
		d.Auth.ID,
	)

	if err != nil {
		return uapi.HttpResponse{
			Status: http.StatusBadRequest,
			Json:   types.ApiError{Message: err.Error()},
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
