package get_announcements

import (
	"net/http"
	"strings"

	"github.com/infinitybotlist/popplio/api"
	"github.com/infinitybotlist/popplio/docs"
	"github.com/infinitybotlist/popplio/state"
	"github.com/infinitybotlist/popplio/types"
	"github.com/infinitybotlist/popplio/utils"

	"github.com/georgysavva/scany/v2/pgxscan"
)

var (
	announcementColsArr = utils.GetCols(types.Announcement{})
	announcementCols    = strings.Join(announcementColsArr, ",")
)

func Docs() *docs.Doc {
	return docs.Route(&docs.Doc{
		Method:      "GET",
		Path:        "/announcements",
		OpId:        "get_announcements",
		Summary:     "Get Announcements",
		Description: "This endpoint will return a list of announcements. User authentication is optional and using it will show user targetted announcements.",
		Tags:        []string{api.CurrentTag},
		Resp:        types.AnnouncementList{},
		AuthType:    []types.TargetType{types.TargetTypeUser},
	})
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

	var target types.UserID

	if d.Auth.Authorized {
		target = types.UserID{
			UserID: d.Auth.ID,
		}
	} else {
		target = types.UserID{}
	}

	annList := []types.Announcement{}

	for _, announcement := range announcements {
		if announcement.Status == "private" {
			// Staff only
			continue
		}

		if announcement.Targetted {
			// Check auth header
			if target.UserID != announcement.Target.String {
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
