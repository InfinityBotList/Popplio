package types

import "popplio/perms"

// rpcPermissions is the staff permission each RPC method requires.
//
// The old model gave every method its own permission, `rpc.` + the method name,
// which meant a role that could approve bots had to be granted five permissions
// and then have the dangerous ones negated back off. Methods that are only ever
// granted together now share one permission, and the ones worth withholding on
// their own — force removal, premium, vote resets — keep theirs.
var rpcPermissions = map[string]perms.Perm{
	"Claim":    perms.StaffReviewBots,
	"Unclaim":  perms.StaffReviewBots,
	"Approve":  perms.StaffReviewBots,
	"Deny":     perms.StaffReviewBots,
	"Unverify": perms.StaffReviewBots,

	"CertifyAdd":    perms.StaffCertifyBots,
	"CertifyRemove": perms.StaffCertifyBots,

	"BotTransferOwnershipUser": perms.StaffTransferBots,
	"BotTransferOwnershipTeam": perms.StaffTransferBots,

	"ForceRemove": perms.StaffForceRemoveBots,

	"PremiumAdd":    perms.StaffManagePremium,
	"PremiumRemove": perms.StaffManagePremium,

	"VoteBanAdd":    perms.StaffBanVoters,
	"VoteBanRemove": perms.StaffBanVoters,

	"VoteReset":    perms.StaffManageVotes,
	"VoteResetAll": perms.StaffManageVotes,

	"AppBanUser":   perms.StaffBanAppUsers,
	"AppUnbanUser": perms.StaffBanAppUsers,
}

// RPCPermission is the permission needed to run the named method. The second
// return is false for a method name that does not exist.
func RPCPermission(name string) (perms.Perm, bool) {
	p, ok := rpcPermissions[name]
	return p, ok
}

// Permission is the permission needed to run this method.
func (m RPCMethod) Permission() (perms.Perm, bool) {
	return RPCPermission(m.Name())
}
