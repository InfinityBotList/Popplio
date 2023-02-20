package current_status

import (
	"net/http"
	"popplio/api"
	"popplio/docs"

	jsoniter "github.com/json-iterator/go"
)

type ListStatus struct {
	StatusPage StatusSummary `json:"status_page"` // From https://status.botlist.site/summary.json
	ApiStatus  ApiStatus     `json:"api_status"`
}

type StatusSummary struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	Status string `json:"status"`
}

type ApiStatus struct {
	ApiUp bool `json:"api_up"`
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Current Status",
		Description: "Gets the current status of the list",
		Resp:        ListStatus{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var listStatus ListStatus

	res, err := http.Get("https://status.botlist.site/summary.json")

	if err != nil {
		listStatus.StatusPage = StatusSummary{
			Name:   "Infinity Bots",
			Url:    "https://status.botlist.site",
			Status: "STATUSPAGEDOWN",
		}
	}

	if res.StatusCode != 200 {
		listStatus.StatusPage = StatusSummary{
			Name:   "Infinity Bots",
			Url:    "https://status.botlist.site",
			Status: "STATUSPAGEDOWN",
		}
	}

	var sp struct {
		Page StatusSummary `json:"page"`
	}

	err = json.NewDecoder(res.Body).Decode(&sp)

	if err != nil {
		listStatus.StatusPage = StatusSummary{
			Name:   "Infinity Bots",
			Url:    "https://status.botlist.site",
			Status: "STATUSPAGEBADRESP",
		}
	}

	listStatus.StatusPage = sp.Page

	listStatus.ApiStatus = ApiStatus{
		ApiUp: true,
	}

	return api.HttpResponse{
		Json: listStatus,
	}
}
