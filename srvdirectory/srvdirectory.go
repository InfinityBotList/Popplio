package srvdirectory

import (
	"popplio/types"
)

var Directory map[string][]types.SDService

func Setup() {
	Directory = map[string][]types.SDService{
		"public": {
			{
				ID:          "htmlsanitize",
				ProdURL:     "https://hs.infinitybots.gg",
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
				ID:          "persepolis",
				ProdURL:     "https://persepolis.infinitybots.gg",
				Description: "Staff onboarding",
			},
		},
	}

}
