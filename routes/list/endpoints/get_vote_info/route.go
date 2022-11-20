package get_vote_info

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/types"
	"popplio/utils"
)

func Docs() {
	docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/vote-info",
		OpId:        "get_vote_info",
		Summary:     "Get Vote Info",
		Description: "Returns basic voting info such as if its a weekend double vote.",
		Resp:        types.VoteInfo{Weekend: true},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var payload = types.VoteInfo{
		Weekend: utils.GetDoubleVote(),
	}

	d.Resp <- types.HttpResponse{
		Json: payload,
	}
}
