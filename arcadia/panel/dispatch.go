package panel

import (
	"context"
	"net/http"

	"popplio/arcadia/types"
)

// dispatch routes a decoded PanelQuery to its handler.
//
// Almost every operation begins by validating login_token. Two validators exist
// and which one an operation uses is load-bearing: CheckAuth requires an ACTIVE
// session, CheckAuthInsecure accepts a pending one (that is how the MFA
// enrolment flow works before a session is activated).
func (s *Server) dispatch(ctx context.Context, req *types.PanelQuery) (response, error) {
	switch {
	case req.Authorize != nil:
		return s.authorize(ctx, req.Authorize)
	case req.Hello != nil:
		return s.hello(ctx, req.Hello)
	case req.BaseAnalytics != nil:
		return s.baseAnalytics(ctx, req.BaseAnalytics)
	case req.GetUser != nil:
		return s.getUser(ctx, req.GetUser)
	case req.BotQueue != nil:
		return s.botQueue(ctx, req.BotQueue)
	case req.ExecuteRpc != nil:
		return s.executeRpc(ctx, req.ExecuteRpc)
	case req.GetRpcMethods != nil:
		return s.getRpcMethods(ctx, req.GetRpcMethods)
	case req.GetRpcLogEntries != nil:
		return s.getRpcLogEntries(ctx, req.GetRpcLogEntries)
	case req.SearchEntitys != nil:
		return s.searchEntitys(ctx, req.SearchEntitys)
	case req.UpdatePartners != nil:
		return s.updatePartners(ctx, req.UpdatePartners)
	case req.UpdateChangelog != nil:
		return s.updateChangelog(ctx, req.UpdateChangelog)
	case req.UpdateBlog != nil:
		return s.updateBlog(ctx, req.UpdateBlog)
	case req.UpdateStaffPositions != nil:
		return s.updateStaffPositions(ctx, req.UpdateStaffPositions)
	case req.UpdateStaffMembers != nil:
		return s.updateStaffMembers(ctx, req.UpdateStaffMembers)
	case req.UpdateStaffDisciplinaryType != nil:
		return s.updateStaffDisciplinaryType(ctx, req.UpdateStaffDisciplinaryType)
	case req.UpdateVoteCreditTiers != nil:
		return s.updateVoteCreditTiers(ctx, req.UpdateVoteCreditTiers)
	case req.UpdateShopItems != nil:
		return s.updateShopItems(ctx, req.UpdateShopItems)
	case req.UpdateShopItemBenefits != nil:
		return s.updateShopItemBenefits(ctx, req.UpdateShopItemBenefits)
	case req.UpdateShopCoupons != nil:
		return s.updateShopCoupons(ctx, req.UpdateShopCoupons)
	case req.UpdateBotWhitelist != nil:
		return s.updateBotWhitelist(ctx, req.UpdateBotWhitelist)
	case req.PopplioStaff != nil:
		return s.popplioStaff(ctx, req.PopplioStaff)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No operation was specified")
	}
}
