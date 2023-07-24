package srvdirectory

import (
	"popplio/state"
	"popplio/types"
)

var Directory map[string][]types.SDService

func Setup() {
	Directory = map[string][]types.SDService{
		"public": {
			{
				ID:          "htmlsanitize",
				Url:         state.Config.Sites.HtmlSanitize,
				Description: "HTML->MD",
			},
			{
				ID:          "popplio",
				Description: "Core API",
				Docs:        "/openapi",
			},
		},
		"staff": {
			{
				ID:           "arcadia",
				Url:          "https://rpc.infinitybots.gg",
				Docs:         "/openapi",
				Description:  "Staff RPC API",
				NeedsStaging: true,
			},
			{
				ID:          "persepolis",
				Url:         "https://persepolis.infinitybots.gg",
				Description: "Responsible for handling onboarding of staff",
			},
			{
				ID:           "ashfur",
				Url:          "https://ashfur.infinitybots.gg",
				Description:  "Responsible for handling data aggregation (modcases) on MongoDB",
				NeedsStaging: true,
			},
		},
	}

}
