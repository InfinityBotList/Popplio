// Package get_list_team implements GET /list/team — "Get List Team".
//
// Gets an up to date listing of the staff team of the list. This is
// currently broken and does not handle permissions yet (TODO)
package get_list_team

import (
	"net/http"
	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	staffMemberCols   = strings.Join(db.GetCols(types.StaffMember{}), ",")
	staffPositionCols = strings.Join(db.GetCols(types.StaffPosition{}), ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get List Team",
		Description: "Gets an up to date listing of the staff team of the list. This is currently broken and does not handle permissions yet (TODO)",
		Resp:        types.StaffTeam{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	sms, err := state.Pool.Query(d.Context, "SELECT "+staffMemberCols+" FROM staff_members")

	if err != nil {
		return resp.Err("Failed to fetch staff team [sms]", err)
	}

	staffMembers, err := pgx.CollectRows(sms, pgx.RowToStructByName[types.StaffMember])

	if err != nil {
		return resp.Err("Failed to fetch staff team [collect]", err)
	}

	var posCache = map[[16]byte]types.StaffPosition{}

	for i, staffMember := range staffMembers {
		user, err := dovewing.GetUser(d.Context, staffMember.ID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch staff team member [dovewing]", err, zap.String("id", staffMember.ID))
		}

		staffMembers[i].User = user
		staffMembers[i].Positions = make([]types.StaffPosition, len(staffMember.PositionIDs))

		for j, position := range staffMember.PositionIDs {
			if pos, ok := posCache[position.Bytes]; ok {
				staffMembers[i].Positions[j] = pos
				continue
			}

			row, err := state.Pool.Query(d.Context, "SELECT "+staffPositionCols+" FROM staff_positions WHERE id = $1", position)

			if err != nil {
				return resp.Err("Failed to fetch staff position [pos]", err, zap.Any("id", position.Bytes))
			}

			pos, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.StaffPosition])

			if err != nil {
				return resp.Err("Failed to fetch staff position [collect]", err, zap.Any("id", position.Bytes))
			}

			staffMembers[i].Positions[j] = pos

			posCache[position.Bytes] = pos
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json: types.StaffTeam{
			Members: staffMembers,
		},
	}
}
