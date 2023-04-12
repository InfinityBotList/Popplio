package get_vote_info

import (
	"net/http"

	"popplio/api"
	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Vote Info",
		Description: "Returns basic voting info such as if its a weekend double vote.",
		Resp:        types.VoteInfo{Weekend: true, VoteTime: 6},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	var payload = types.VoteInfo{
		Weekend:  utils.GetDoubleVote(),
		VoteTime: utils.GetVoteTime(),
	}

	return api.HttpResponse{
		Json: payload,
	}
}
