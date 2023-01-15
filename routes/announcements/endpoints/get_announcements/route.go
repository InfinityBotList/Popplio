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
		Method:      "GET",
		Summary:     "Get Announcements",
		Description: "This endpoint will return a list of announcements. User authentication is optional and using it will show user targetted announcements.",
		Resp:        types.AnnouncementList{},
	}
}

func Route(d api.RouteData, r *http.Request) api.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+announcementCols+" FROM announcements ORDER BY id DESC")

	if err != nil {
		state.Logger.Error("Could not", err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	var announcements []types.Announcement

	err = pgxscan.ScanAll(&announcements, rows)

	if err != nil {
		state.Logger.Error(err)
		return api.DefaultResponse(http.StatusNotFound)
	}

	// Auth header check

	annList := []types.Announcement{}

	for _, announcement := range announcements {
		if announcement.Status == "private" {
			// Staff only
			continue
		}

		if announcement.Targetted {
			// Check auth header
			if !d.Auth.Authorized || d.Auth.ID != announcement.Target.String {
				continue
			}
		}

		annList = append(annList, announcement)
	}

	annListObj := types.AnnouncementList{
		Announcements: annList,
	}

	return api.HttpResponse{
		Json: annListObj,
	}
}
