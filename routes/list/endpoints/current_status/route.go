// Package current_status implements GET /list/current-status — "Get Current
// Status".
//
// Gets the current status of the list
package current_status

import (
	"net/http"
	"net/url"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"
	"strings"
	"time"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Current Status",
		Description: "Gets the current status of the list",
		Resp:        types.StatusDocs{},
		Params: []docs.Parameter{
			{
				Name:        "src",
				Description: "Source to use. If unspecified, defaults to instatus",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var listStatus map[string]any

	src := r.URL.Query().Get("src")

	if src == "" {
		src = "instatus"
	}

	// Check if response is on redis
	cachedResp := state.Redis.Get(d.Context, "current_status:"+src)

	if cachedResp.Val() != "" {
		return uapi.HttpResponse{
			Json: cachedResp.Val(),
			Headers: map[string]string{
				"X-Cache": "HIT",
			},
		}
	}

	switch src {
	case "instatus":
		res, err := http.Get(state.Config.Sites.Instatus + "/summary.json")

		if err != nil {
			return resp.ErrBody("Instatus returned an error", "Instatus returned an error.", err)
		}

		if res.StatusCode != 200 {
			return resp.ErrBody("Instatus returned a non-200 status code:", "Instatus returned a non-200 status code: "+res.Status, nil)
		}

		err = jsonimpl.UnmarshalReader(res.Body, &listStatus)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	case "uptime-robot":
		// create form
		form := url.Values{}
		form.Set("api_key", state.Config.Meta.UptimeRobotROAPIKey)
		form.Set("response_times", "1")
		form.Set("custom_uptime_ratios", "7-30")

		// create request
		client := http.Client{
			Timeout: 10 * time.Second,
		}

		req, err := http.NewRequest("POST", "https://api.uptimerobot.com/v2/getMonitors", strings.NewReader(form.Encode()))

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		// set content type
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// make request
		res, err := client.Do(req)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		if res.StatusCode != 200 {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}

		err = jsonimpl.UnmarshalReader(res.Body, &listStatus)

		if err != nil {
			return uapi.DefaultResponse(http.StatusInternalServerError)
		}
	default:
		return resp.BadRequest("Invalid source. Valid sources are instatus and uptime-robot")
	}

	// Cache response
	state.Redis.Set(d.Context, "current_status:"+src, listStatus, 3*time.Minute)

	return uapi.HttpResponse{
		Json: listStatus,
		Headers: map[string]string{
			"X-Cache": "MISS",
		},
	}
}
