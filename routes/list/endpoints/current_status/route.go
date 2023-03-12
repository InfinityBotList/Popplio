package current_status

import (
	"net/http"
	"popplio/api"

	docs "github.com/infinitybotlist/doclib"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Current Status",
		Description: "Gets the current status of the list",
		Resp:        map[string]any{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var listStatus map[string]any

	res, err := http.Get("https://status.botlist.site/summary.json")

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	if res.StatusCode != 200 {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	err = json.NewDecoder(res.Body).Decode(&listStatus)

	if err != nil {
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	return api.HttpResponse{
		Json: listStatus,
	}
}
