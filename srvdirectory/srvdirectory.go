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
				ProdURL:     state.Config.Sites.HtmlSanitize,
				Description: "HTML->MD",
				Docs:        "/openapi",
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
				ProdURL:      "https://prod--panel-api.infinitybots.gg",
				Docs:         "/openapi",
				Description:  "Staff Panel API",
				NeedsStaging: true,
			},
			{
				ID:          "persepolis",
				ProdURL:     "https://persepolis.infinitybots.gg",
				Description: "Staff onboarding",
			},
			{
				ID:           "ashfur",
				ProdURL:      "https://ashfur.infinitybots.gg",
				Description:  "Data aggregation (modcases) on MongoDB",
				NeedsStaging: true,
			},
		},
	}

}
