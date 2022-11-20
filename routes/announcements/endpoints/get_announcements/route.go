package get_announcements

import (
	"net/http"
	"popplio/api"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

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

func Route(d api.RouteData, r *http.Request) {
	rows, err := state.Pool.Query(d.Context, "SELECT "+announcementCols+" FROM announcements ORDER BY id DESC")

	if err != nil {
		state.Logger.Error("Could not", err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
	}

	var announcements []types.Announcement

	err = pgxscan.ScanAll(&announcements, rows)

	if err != nil {
		state.Logger.Error(err)
		d.Resp <- utils.ApiDefaultReturn(http.StatusNotFound)
		return
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

	d.Resp <- types.HttpResponse{
		Json: annListObj,
	}
}
