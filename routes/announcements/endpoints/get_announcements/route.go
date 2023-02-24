package get_announcements

import (
	"net/http"
	"strings"

	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	announcementColsArr = utils.GetCols(types.Announcement{})
	announcementCols    = strings.Join(announcementColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Announcements",
		Description: "Returns the public announcements on the list.",
		Resp:        types.AnnouncementList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+announcementCols+" FROM announcements WHERE status = 'public' ORDER BY id DESC")

	if err != nil {
		state.Logger.Error("Could not load announcements", err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	var announcements = []types.Announcement{}

	err = pgxscan.ScanAll(&announcements, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusInternalServerError)
	}

	for i := range announcements {
		announcements[i].Author, err = utils.GetDiscordUser(d.Context, announcements[i].UserID)

		if err != nil {
			state.Logger.Error(err)
			return api.DefaultResponse(http.StatusInternalServerError)
		}
	}

	return api.HttpResponse{
		Json: types.AnnouncementList{
			Announcements: announcements,
		},
	}
}
