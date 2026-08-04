package perms

// The entity permissions: what a team member may do with the team and the bots
// and servers it owns.
//
// The old model crossed every action with every entity type, so a team had
// separate permissions for getting, creating, editing, deleting and testing
// webhooks on each of bots, servers and the team itself — twenty-one
// permissions for one feature. Actions that are only ever granted together are
// now one permission, and the entity type is only kept where it changes who
// should be able to do something: adding, editing and deleting bots is a
// different decision from doing the same to servers.
const (
	// EntityOwner implies every other entity permission.
	EntityOwner Perm = "owner"

	EntityEditTeam  Perm = "edit_team"
	EntitySetVanity Perm = "set_vanity"

	EntityAddBots             Perm = "add_bots"
	EntityEditBots            Perm = "edit_bots"
	EntityDeleteBots          Perm = "delete_bots"
	EntityResubmitBots        Perm = "resubmit_bots"
	EntityCertifyBots         Perm = "request_bot_certification"
	EntityAddServers          Perm = "add_servers"
	EntityEditServers         Perm = "edit_servers"
	EntityDeleteServers       Perm = "delete_servers"
	EntityManageServerInvites Perm = "manage_server_invites"

	EntityAddMembers    Perm = "add_team_members"
	EntityEditMembers   Perm = "edit_team_members"
	EntityRemoveMembers Perm = "remove_team_members"

	EntityViewWebhooks       Perm = "view_webhooks"
	EntityManageWebhooks     Perm = "manage_webhooks"
	EntityViewWebhookLogs    Perm = "view_webhook_logs"
	EntityDeleteWebhookLogs  Perm = "delete_webhook_logs"
	EntityManageAssets       Perm = "manage_assets"
	EntityManageOwnerReviews Perm = "manage_owner_reviews"

	EntityViewSessions   Perm = "view_sessions"
	EntityManageSessions Perm = "manage_sessions"

	EntityRedeemVoteCredits Perm = "redeem_vote_credits"
)

// Entity is what a team member may do. Teams have no roles, so a member's
// `flags` and an API session's `perm_limits` are the whole source.
var Entity = NewCatalogue("entity", EntityOwner, []Definition{
	{
		ID:          EntityOwner,
		Name:        "Owner",
		Description: "Full control over the team, everything it owns and everyone on it, including deleting the team. This implies every other permission.",
		Category:    "General",
		Dangerous:   true,
		Legacy:      []string{"global.*"},
	},
	{
		ID:          EntityEditTeam,
		Name:        "Edit Team",
		Description: "Change the team's name, description, links and other settings.",
		Category:    "General",
		Legacy:      []string{"team.edit", "global.edit"},
	},
	{
		ID:          EntitySetVanity,
		Name:        "Set Vanity",
		Description: "Change the vanity URL of the team or anything it owns.",
		Category:    "General",
		Legacy:      []string{"team.set_vanity", "bot.set_vanity", "server.set_vanity", "global.set_vanity"},
	},

	{
		ID:          EntityAddBots,
		Name:        "Add Bots",
		Description: "Add a new bot to the team, or transfer one into it.",
		Category:    "Bots",
		Legacy:      []string{"bot.add", "global.add"},
	},
	{
		ID:          EntityEditBots,
		Name:        "Edit Bots",
		Description: "Change the settings of the team's bots.",
		Category:    "Bots",
		Legacy:      []string{"bot.edit"},
	},
	{
		ID:          EntityDeleteBots,
		Name:        "Delete Bots",
		Description: "Remove a bot from the team. This cannot be undone.",
		Category:    "Bots",
		Dangerous:   true,
		Legacy:      []string{"bot.delete", "global.delete"},
	},
	{
		ID:          EntityResubmitBots,
		Name:        "Resubmit Bots",
		Description: "Put a denied bot back into the review queue.",
		Category:    "Bots",
		Legacy:      []string{"bot.resubmit", "global.resubmit"},
	},
	{
		ID:          EntityCertifyBots,
		Name:        "Request Certification",
		Description: "Ask staff to certify one of the team's bots.",
		Category:    "Bots",
		Legacy:      []string{"bot.request_cert", "global.request_cert"},
	},

	{
		ID:          EntityAddServers,
		Name:        "Add Servers",
		Description: "Add a new server to the team, or transfer one into it.",
		Category:    "Servers",
		Legacy:      []string{"server.add"},
	},
	{
		ID:          EntityEditServers,
		Name:        "Edit Servers",
		Description: "Change the settings of the team's servers.",
		Category:    "Servers",
		Legacy:      []string{"server.edit"},
	},
	{
		ID:          EntityDeleteServers,
		Name:        "Delete Servers",
		Description: "Remove a server from the team. This cannot be undone.",
		Category:    "Servers",
		Dangerous:   true,
		Legacy:      []string{"server.delete"},
	},
	{
		ID:          EntityManageServerInvites,
		Name:        "Manage Server Invites",
		Description: "Read and change the invite of the team's servers.",
		Category:    "Servers",
		Legacy:      []string{"server.get_invite", "server.set_invite", "global.get_invite", "global.set_invite"},
	},

	{
		ID:          EntityAddMembers,
		Name:        "Add Members",
		Description: "Invite users onto the team.",
		Category:    "Members",
		Legacy:      []string{"team_member.add"},
	},
	{
		ID:          EntityEditMembers,
		Name:        "Edit Members",
		Description: "Change what other members of the team may do.",
		Category:    "Members",
		Dangerous:   true,
		Legacy:      []string{"team_member.edit"},
	},
	{
		ID:          EntityRemoveMembers,
		Name:        "Remove Members",
		Description: "Remove members from the team.",
		Category:    "Members",
		Legacy:      []string{"team_member.delete", "team_member.remove"},
	},

	{
		ID:          EntityViewWebhooks,
		Name:        "View Webhooks",
		Description: "See the webhooks configured on the team and what it owns.",
		Category:    "Webhooks",
		Legacy:      []string{"bot.get_webhooks", "server.get_webhooks", "team.get_webhooks", "global.get_webhooks"},
	},
	{
		ID:          EntityManageWebhooks,
		Name:        "Manage Webhooks",
		Description: "Create, edit, test and delete webhooks.",
		Category:    "Webhooks",
		Legacy: []string{
			"bot.create_webhooks", "server.create_webhooks", "team.create_webhooks", "global.create_webhooks",
			"bot.edit_webhooks", "server.edit_webhooks", "team.edit_webhooks", "global.edit_webhooks",
			"bot.delete_webhooks", "server.delete_webhooks", "team.delete_webhooks", "global.delete_webhooks",
			"bot.test_webhooks", "server.test_webhooks", "team.test_webhooks", "global.test_webhooks",
		},
	},
	{
		ID:          EntityViewWebhookLogs,
		Name:        "View Webhook Logs",
		Description: "Read the delivery logs of the team's webhooks.",
		Category:    "Webhooks",
		Legacy:      []string{"bot.get_webhook_logs", "server.get_webhook_logs", "team.get_webhook_logs", "global.get_webhook_logs"},
	},
	{
		ID:          EntityDeleteWebhookLogs,
		Name:        "Delete Webhook Logs",
		Description: "Clear the delivery logs of the team's webhooks.",
		Category:    "Webhooks",
		Legacy: []string{
			"bot.delete_webhook_logs", "server.delete_webhook_logs",
			"team.delete_webhook_logs", "global.delete_webhook_logs",
		},
	},

	{
		ID:          EntityManageAssets,
		Name:        "Manage Assets",
		Description: "Upload and delete avatars, banners and other assets.",
		Category:    "Content",
		Legacy: []string{
			"bot.upload_assets", "server.upload_assets", "team.upload_assets", "global.upload_assets",
			"bot.delete_assets", "server.delete_assets", "team.delete_assets", "global.delete_assets",
		},
	},
	{
		ID:          EntityManageOwnerReviews,
		Name:        "Manage Owner Reviews",
		Description: "Write, edit and delete the team's owner reviews.",
		Category:    "Content",
		Legacy: []string{
			"bot.create_owner_review", "server.create_owner_review", "team.create_owner_review", "global.create_owner_review",
			"bot.edit_owner_review", "server.edit_owner_review", "team.edit_owner_review", "global.edit_owner_review",
			"bot.delete_owner_review", "server.delete_owner_review", "team.delete_owner_review", "global.delete_owner_review",
		},
	},

	{
		ID:          EntityViewSessions,
		Name:        "View API Sessions",
		Description: "See the API sessions that exist on the team and what it owns.",
		Category:    "API Access",
		Legacy: []string{
			"bot.view_session", "server.view_session", "team.view_session", "global.view_session",
			"bot.view_api_tokens", "server.view_api_tokens", "team.view_api_tokens", "global.view_api_tokens",
		},
	},
	{
		ID:          EntityManageSessions,
		Name:        "Manage API Sessions",
		Description: "Create and revoke API sessions, which grant programmatic access to the entity.",
		Category:    "API Access",
		Dangerous:   true,
		Legacy: []string{
			"bot.create_session", "server.create_session", "team.create_session", "global.create_session",
			"bot.revoke_session", "server.revoke_session", "team.revoke_session", "global.revoke_session",
			"bot.reset_api_tokens", "server.reset_api_tokens", "team.reset_api_tokens", "global.reset_api_tokens",
		},
	},

	{
		ID:          EntityRedeemVoteCredits,
		Name:        "Redeem Vote Credits",
		Description: "Redeem the vote credits the team's entities have earned.",
		Category:    "Votes",
		Legacy: []string{
			"bot.redeem_vote_credits", "server.redeem_vote_credits",
			"team.redeem_vote_credits", "global.redeem_vote_credits",
		},
	},
})

// EntityLifecycle maps a target type to the permissions that govern adding,
// editing and deleting one of them, so routes that work on "whatever the URL
// points at" do not each need their own switch.
func EntityLifecycle(targetType string) (add, edit, del Perm, ok bool) {
	switch targetType {
	case "bot":
		return EntityAddBots, EntityEditBots, EntityDeleteBots, true
	case "server":
		return EntityAddServers, EntityEditServers, EntityDeleteServers, true
	case "team":
		// A team is created by its owner and deleted with Owner, so the only
		// lifecycle permission it has is editing.
		return EntityEditTeam, EntityEditTeam, EntityOwner, true
	default:
		return "", "", "", false
	}
}
