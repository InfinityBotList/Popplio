package get_vote_info

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/utils"
)

type VoteInfo struct {
	Weekend  bool   `json:"is_weekend"`
	VoteTime uint16 `json:"vote_time"`
}

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/list/vote-info",
		OpId:        "get_vote_info",
		Summary:     "Get Vote Info",
		Description: "Returns basic voting info such as if its a weekend double vote.",
		Resp:        VoteInfo{Weekend: true, VoteTime: 6},
		Tags:        []string{api.CurrentTag},
	})
}

func Route(d api.RouteData, r *http.Request) {
	var payload = VoteInfo{
		Weekend:  utils.GetDoubleVote(),
		VoteTime: utils.GetVoteTime(),
	}

	d.Resp <- api.HttpResponse{
		Json: payload,
	}
}
