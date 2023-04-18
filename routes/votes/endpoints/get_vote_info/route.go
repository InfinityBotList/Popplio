package get_vote_info

import (
	"net/http"

	"popplio/types"
	"popplio/utils"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Vote Info",
		Description: "Returns basic voting info such as if its a weekend double vote.",
		Resp:        types.VoteInfo{Weekend: true, VoteTime: 6},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var payload = types.VoteInfo{
		Weekend:  utils.GetDoubleVote(),
		VoteTime: utils.GetVoteTime(),
	}

	return uapi.HttpResponse{
		Json: payload,
	}
}
