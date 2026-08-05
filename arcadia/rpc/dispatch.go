package rpc

import (
	"context"
	"errors"

	"popplio/arcadia/types"
)

// This file is the one place that says which handler runs for which method.
//
// Adding an action is three edits and no more: a variant on types.RPCMethod, a
// case here, and the handler itself in the file for its area. Nothing else in
// the package needs to know the new action exists.

// handleMethod is the low-level dispatcher: it picks the handler for whichever
// variant the method carries.
//
// It runs after Execute has checked the target type, the permission and the rate
// limit, so a handler can assume the caller was allowed to reach it.
func handleMethod(ctx context.Context, method types.RPCMethod, h Handle) (Success, error) {
	switch {
	case method.Claim != nil:
		return claim(ctx, method.Claim, h)
	case method.Unclaim != nil:
		return unclaim(ctx, method.Unclaim, h)
	case method.Approve != nil:
		return approve(ctx, method.Approve, h)
	case method.Deny != nil:
		return deny(ctx, method.Deny, h)
	case method.Unverify != nil:
		return unverify(ctx, method.Unverify, h)
	case method.PremiumAdd != nil:
		return premiumAdd(ctx, method.PremiumAdd, h)
	case method.PremiumRemove != nil:
		return premiumRemove(ctx, method.PremiumRemove, h)
	case method.VoteBanAdd != nil:
		return voteBanSet(ctx, method.VoteBanAdd, h, true)
	case method.VoteBanRemove != nil:
		return voteBanSet(ctx, method.VoteBanRemove, h, false)
	case method.VoteReset != nil:
		return voteReset(ctx, method.VoteReset, h)
	case method.VoteResetAll != nil:
		return voteResetAll(ctx, method.VoteResetAll, h)
	case method.ForceRemove != nil:
		return forceRemove(ctx, method.ForceRemove, h)
	case method.CertifyAdd != nil:
		return certifyAdd(ctx, method.CertifyAdd, h)
	case method.CertifyRemove != nil:
		return certifyRemove(ctx, method.CertifyRemove, h)
	case method.BotTransferOwnershipUser != nil:
		return transferOwnershipUser(ctx, method.BotTransferOwnershipUser, h)
	case method.BotTransferOwnershipTeam != nil:
		return transferOwnershipTeam(ctx, method.BotTransferOwnershipTeam, h)
	case method.AppBanUser != nil:
		return appBanSet(ctx, method.AppBanUser, h, true)
	case method.AppUnbanUser != nil:
		return appBanSet(ctx, method.AppUnbanUser, h, false)
	default:
		return Success{}, errors.New("This method does not support this target type yet")
	}
}
