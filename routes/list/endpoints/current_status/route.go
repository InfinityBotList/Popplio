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
	"go.uber.org/zap"
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

	httpClient := http.Client{Timeout: 10 * time.Second}

	switch src {
	case "instatus":
		req, err := http.NewRequestWithContext(d.Context, http.MethodGet, state.Config.Sites.Instatus+"/summary.json", nil)

		if err != nil {
			return resp.Err("Error while building Instatus request", err)
		}

		res, err := httpClient.Do(req)

		if err != nil {
			return resp.Err("Instatus returned an error", err)
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			return resp.Err("Instatus returned a non-200 status code", nil, zap.String("status", res.Status))
		}

		err = jsonimpl.UnmarshalReader(res.Body, &listStatus)

		if err != nil {
			return resp.Err("Error while unmarshalling Instatus response", err)
		}
	case "uptime-robot":
		// create form
		form := url.Values{}
		form.Set("api_key", state.Config.Meta.UptimeRobotROAPIKey)
		form.Set("response_times", "1")
		form.Set("custom_uptime_ratios", "7-30")

		// create request
		req, err := http.NewRequestWithContext(d.Context, http.MethodPost, "https://api.uptimerobot.com/v2/getMonitors", strings.NewReader(form.Encode()))

		if err != nil {
			return resp.Err("Error while building UptimeRobot request", err)
		}

		// set content type
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// make request
		res, err := httpClient.Do(req)

		if err != nil {
			return resp.Err("UptimeRobot returned an error", err)
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			return resp.Err("UptimeRobot returned a non-200 status code", nil, zap.String("status", res.Status))
		}

		err = jsonimpl.UnmarshalReader(res.Body, &listStatus)

		if err != nil {
			return resp.Err("Error while unmarshalling UptimeRobot response", err)
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
