package current_status

import (
	"net/http"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Current Status",
		Description: "Gets the current status of the list",
		Resp:        types.StatusDocs{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var listStatus map[string]any

	res, err := http.Get("https://status.botlist.site/summary.json")

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	if res.StatusCode != 200 {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	err = json.NewDecoder(res.Body).Decode(&listStatus)

	if err != nil {
		return uapi.DefaultResponse(http.StatusInternalServerError)
	}

	return uapi.HttpResponse{
		Json: listStatus,
	}
}
