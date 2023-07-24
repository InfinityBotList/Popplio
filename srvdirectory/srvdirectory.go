package srvdirectory

import (
	"popplio/state"
	"popplio/types"
)

var Directory map[string]map[string]types.SDService

func Setup() {
	Directory = map[string]map[string]types.SDService{
		"public": {
			"htmlsanitize": {
				Url:         state.Config.Sites.HtmlSanitize,
				Description: "HTML->MD",
			},
			"popplio": {
				Description: "Core API",
				Docs:        "/openapi",
			},
		},
		"staff": {
			"arcadia": {
				Url:          "https://rpc.infinitybots.gg",
				Docs:         "/openapi",
				Description:  "Staff RPC API",
				NeedsStaging: true,
			},
			"persepolis": {
				Url:         "https://persepolis.infinitybots.gg",
				Description: "Responsible for handling onboarding of staff",
			},
			"ashfur": {
				Url:          "https://ashfur.infinitybots.gg",
				Description:  "Responsible for handling data aggregation (modcases) on MongoDB",
				NeedsStaging: true,
			},
		},
	}

}
