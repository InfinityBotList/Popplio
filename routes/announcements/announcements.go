package announcements

import (
	"net/http"
	"popplio/docs"
	"popplio/state"
	"popplio/types"
	"popplio/utils"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

const tagName = "Announcements"

var (
	announcementColsArr = utils.GetCols(types.Announcement{})
	announcementCols    = strings.Join(announcementColsArr, ",")
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to our announcements system"
}

func (b Router) Routes(r *chi.Mux) {
	r.Route("/announcements", func(r chi.Router) {
		docs.Route(&docs.Doc{
			Method:      "GET",
			Path:        "/announcements",
			OpId:        "announcements",
			Summary:     "Get Announcements",
			Description: "This endpoint will return a list of announcements. User authentication is optional and using it will show user targetted announcements.",
			Tags:        []string{tagName},
			Resp:        types.AnnouncementList{},
			AuthType:    []string{"User"},
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			resp := make(chan types.HttpResponse)

			go func() {
				rows, err := state.Pool.Query(ctx, "SELECT "+announcementCols+" FROM announcements ORDER BY id DESC")

				if err != nil {
					state.Logger.Error("Could not", err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				var announcements []types.Announcement

				err = pgxscan.ScanAll(&announcements, rows)

				if err != nil {
					state.Logger.Error(err)
					resp <- utils.ApiDefaultReturn(http.StatusNotFound)
					return
				}

				// Auth header check
				auth := r.Header.Get("Authorization")

				var target types.UserID

				if auth != "" {
					targetId := utils.AuthCheck(auth, false)

					if targetId != nil {
						state.Logger.Error(err)
						resp <- utils.ApiDefaultReturn(http.StatusUnauthorized)
						return
					}

					target = types.UserID{UserID: *targetId}
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

				resp <- types.HttpResponse{
					Json: annListObj,
				}
			}()

			utils.Respond(ctx, w, resp)
		})
	})
}
