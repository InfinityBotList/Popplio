package perms

// The staff permissions.
//
// Names are verb_object and describe what someone may do, not which internal
// endpoint they may reach. Where the old model had one permission per RPC
// method, related methods now share one permission: `review_bots` covers the
// whole claim → approve/deny loop, because no role has ever held one of those
// without the others.
const (
	// StaffAdministrator implies every other staff permission.
	StaffAdministrator Perm = "administrator"

	StaffViewPanel     Perm = "view_panel"
	StaffViewAuditLogs Perm = "view_audit_logs"
	StaffViewSensitive Perm = "view_sensitive_data"
	StaffUseStagingKey Perm = "use_staging_keys"

	StaffReviewBots      Perm = "review_bots"
	StaffCertifyBots     Perm = "certify_bots"
	StaffTransferBots    Perm = "transfer_bots"
	StaffForceRemoveBots Perm = "force_remove_bots"

	StaffManagePremium Perm = "manage_premium"
	StaffManageVotes   Perm = "manage_votes"
	StaffBanVoters     Perm = "ban_voters"

	StaffViewApps    Perm = "view_apps"
	StaffManageApps  Perm = "manage_apps"
	StaffBanAppUsers Perm = "ban_app_users"

	StaffViewStaff            Perm = "view_staff"
	StaffManageStaffMembers   Perm = "manage_staff_members"
	StaffManageStaffRoles     Perm = "manage_staff_roles"
	StaffManageDisciplinaries Perm = "manage_disciplinaries"
	StaffViewOnboarding       Perm = "view_onboarding"

	StaffViewShop           Perm = "view_shop"
	StaffManageShop         Perm = "manage_shop"
	StaffManageBotWhitelist Perm = "manage_bot_whitelist"

	StaffManagePartners Perm = "manage_partners"
	StaffManageBlog     Perm = "manage_blog"

	StaffViewTickets Perm = "view_tickets"

	StaffViewCDN   Perm = "view_cdn"
	StaffManageCDN Perm = "manage_cdn"

	// The marker permissions carry no power inside Popplio. They label a staff
	// member for other Omniplex services and for role display.
	StaffMarkerDeveloper      Perm = "marker_developer"
	StaffMarkerLeadDeveloper  Perm = "marker_lead_developer"
	StaffMarkerHumanResources Perm = "marker_human_resources"
	StaffMarkerBotReviewer    Perm = "marker_bot_reviewer"
	StaffMarkerServiceAccount Perm = "marker_service_account"
	StaffMarkerDisciplinary   Perm = "marker_disciplinary"
)

// Staff is what a staff member may do on the platform. Its roles are the
// `staff_positions` rows, each bound to a Discord role in the staff server.
var Staff = NewCatalogue("staff", StaffAdministrator, []Definition{
	{
		ID:          StaffAdministrator,
		Name:        "Administrator",
		Description: "Full control over everything. This implies every other staff permission and should be held by as few people as possible.",
		Category:    "Administration",
		Dangerous:   true,
		Legacy:      []string{"global.*", "arcadia.*"},
	},
	{
		ID:          StaffViewPanel,
		Name:        "View Panel",
		Description: "Access the staff panel and see the data on it.",
		Category:    "Administration",
		Legacy:      []string{"global.view"},
	},
	{
		ID:          StaffViewAuditLogs,
		Name:        "View Audit Logs",
		Description: "See the log of every staff action taken through the panel and the staff bot.",
		Category:    "Administration",
		Legacy:      []string{"rpc_logs.view", "rpc_logs.*"},
	},
	{
		ID:          StaffViewSensitive,
		Name:        "View Sensitive Data",
		Description: "See data hidden from other staff, such as private contact details on an entity.",
		Category:    "Administration",
		Dangerous:   true,
		Legacy:      []string{"global.view_sensitive"},
	},
	{
		ID:          StaffUseStagingKey,
		Name:        "Use Staging Keys",
		Description: "Perform actions that use test payment keys on staging and development instances.",
		Category:    "Administration",
		Legacy:      []string{"popplio_staging.sensitive", "popplio_staging.*"},
	},

	{
		ID:          StaffReviewBots,
		Name:        "Review Bots",
		Description: "Claim, unclaim, approve, deny and unverify bots in the review queue.",
		Category:    "Bot Reviews",
		Legacy:      []string{"rpc.Claim", "rpc.Unclaim", "rpc.Approve", "rpc.Deny", "rpc.Unverify"},
	},
	{
		ID:          StaffCertifyBots,
		Name:        "Certify Bots",
		Description: "Grant and remove certification on a bot.",
		Category:    "Bot Reviews",
		Legacy:      []string{"rpc.CertifyAdd", "rpc.CertifyRemove"},
	},
	{
		ID:          StaffTransferBots,
		Name:        "Transfer Bots",
		Description: "Move a bot to a different owner or team.",
		Category:    "Bot Reviews",
		Legacy:      []string{"rpc.BotTransferOwnershipUser", "rpc.BotTransferOwnershipTeam"},
	},
	{
		ID:          StaffForceRemoveBots,
		Name:        "Force Remove Bots",
		Description: "Delete a bot from the list outright. This cannot be undone.",
		Category:    "Bot Reviews",
		Dangerous:   true,
		Legacy:      []string{"rpc.ForceRemove"},
	},

	{
		ID:          StaffManagePremium,
		Name:        "Manage Premium",
		Description: "Give and take premium status on an entity.",
		Category:    "Users & Votes",
		Dangerous:   true,
		Legacy:      []string{"rpc.PremiumAdd", "rpc.PremiumRemove"},
	},
	{
		ID:          StaffManageVotes,
		Name:        "Manage Votes",
		Description: "Reset the votes of an entity, or of every entity at once.",
		Category:    "Users & Votes",
		Dangerous:   true,
		Legacy:      []string{"rpc.VoteReset", "rpc.VoteResetAll"},
	},
	{
		ID:          StaffBanVoters,
		Name:        "Ban Voters",
		Description: "Vote ban and unban a user.",
		Category:    "Users & Votes",
		Legacy:      []string{"rpc.VoteBanAdd", "rpc.VoteBanRemove"},
	},

	{
		ID:          StaffViewApps,
		Name:        "View Applications",
		Description: "Read staff and partner applications.",
		Category:    "Applications",
		Legacy:      []string{"apps.view"},
	},
	{
		ID:          StaffManageApps,
		Name:        "Manage Applications",
		Description: "Approve, deny and otherwise act on applications.",
		Category:    "Applications",
		Legacy:      []string{"apps.manage"},
	},
	{
		ID:          StaffBanAppUsers,
		Name:        "Ban Applicants",
		Description: "Bar a user from submitting further applications, and lift that ban.",
		Category:    "Applications",
		Legacy:      []string{"rpc.AppBanUser", "rpc.AppUnbanUser"},
	},

	{
		ID:          StaffViewStaff,
		Name:        "View Staff",
		Description: "See the staff list, the roles that exist and who holds them.",
		Category:    "Staff Management",
		Legacy:      []string{"staff_members.view", "staff_positions.view"},
	},
	{
		ID:          StaffManageStaffMembers,
		Name:        "Manage Staff Members",
		Description: "Edit a staff member's extra permissions and their sync settings.",
		Category:    "Staff Management",
		Dangerous:   true,
		Legacy:      []string{"staff_members.edit"},
	},
	{
		ID:          StaffManageStaffRoles,
		Name:        "Manage Staff Roles",
		Description: "Create, edit, reorder and delete staff roles and the permissions attached to them.",
		Category:    "Staff Management",
		Dangerous:   true,
		Legacy: []string{
			"staff_positions.create", "staff_positions.edit", "staff_positions.delete",
			"staff_positions.set_index", "staff_positions.swap_index",
		},
	},
	{
		ID:          StaffManageDisciplinaries,
		Name:        "Manage Disciplinaries",
		Description: "Create, edit and delete the disciplinary types that limit a staff member's permissions.",
		Category:    "Staff Management",
		Dangerous:   true,
		Legacy: []string{
			"staff_disciplinary_types.create", "staff_disciplinary_types.update",
			"staff_disciplinary_types.delete", "staff_disciplinary_types.*",
		},
	},
	{
		ID:          StaffViewOnboarding,
		Name:        "View Onboarding",
		Description: "Read the onboarding responses submitted by new staff.",
		Category:    "Staff Management",
		Legacy:      []string{"persepolis.view_onboarding_responses", "persepolis.*"},
	},

	{
		ID:          StaffViewShop,
		Name:        "View Shop",
		Description: "See shop items, benefits, coupons and vote credit tiers.",
		Category:    "Shop",
		Legacy:      []string{"shop.view", "shop_items.view", "shop_item_benefits.view", "shop_coupons.list"},
	},
	{
		ID:          StaffManageShop,
		Name:        "Manage Shop",
		Description: "Create, edit and delete shop items, benefits, coupons and vote credit tiers.",
		Category:    "Shop",
		Legacy: []string{
			"shop_items.create", "shop_items.update", "shop_items.delete",
			"shop_item_benefits.create", "shop_item_benefits.update", "shop_item_benefits.delete",
			"shop_coupons.create", "shop_coupons.update", "shop_coupons.delete",
			"vote_credit_tiers.create", "vote_credit_tiers.update", "vote_credit_tiers.delete",
		},
	},
	{
		ID:          StaffManageBotWhitelist,
		Name:        "Manage Bot Whitelist",
		Description: "Control which bots are whitelisted for shop purchases.",
		Category:    "Shop",
		Legacy: []string{
			"bot_whitelist.create", "bot_whitelist.update", "bot_whitelist.delete", "bot_whitelist.*",
		},
	},

	{
		ID:          StaffManagePartners,
		Name:        "Manage Partners",
		Description: "Create, edit and delete partners.",
		Category:    "Content",
		Legacy:      []string{"partners.create", "partners.update", "partners.delete", "partners.*"},
	},
	{
		ID:          StaffManageBlog,
		Name:        "Manage Blog",
		Description: "Create, edit and delete blog entries.",
		Category:    "Content",
		Legacy:      []string{"blog.create_entry", "blog.update_entry", "blog.delete_entry", "blog.*"},
	},

	{
		ID:          StaffViewTickets,
		Name:        "View Tickets",
		Description: "Read support tickets opened by users.",
		Category:    "Support",
		Legacy:      []string{"popplio.tickets", "popplio.*"},
	},

	{
		ID:          StaffViewCDN,
		Name:        "View CDN",
		Description: "List CDN scopes and read the files in them.",
		Category:    "External Services",
		Legacy:      []string{"cdn.list_scopes", "cdn.list", "cdn#ibl@main.list", "cdn#ibl@main.read_file"},
	},
	{
		ID:          StaffManageCDN,
		Name:        "Manage CDN",
		Description: "Upload, replace and delete files on the CDN.",
		Category:    "External Services",
		Dangerous:   true,
		Legacy:      []string{"cdn.add_file", "cdn.upload_chunk"},
	},
	{
		ID:          StaffMarkerDeveloper,
		Name:        "Developer",
		Description: "Marks the holder as a developer. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"developer.marker"},
	},
	{
		ID:          StaffMarkerLeadDeveloper,
		Name:        "Lead Developer",
		Description: "Marks the holder as a lead developer. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"lead_developer.marker"},
	},
	{
		ID:          StaffMarkerHumanResources,
		Name:        "Human Resources",
		Description: "Marks the holder as human resources. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"human_resources.marker"},
	},
	{
		ID:          StaffMarkerBotReviewer,
		Name:        "Bot Reviewer",
		Description: "Marks the holder as a bot reviewer. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"bot_reviewer.marker"},
	},
	{
		ID:          StaffMarkerServiceAccount,
		Name:        "Service Account",
		Description: "Marks the holder as a service account rather than a person. Carries no power on its own.",
		Category:    "Markers",
		Legacy:      []string{"service_account.marker"},
	},
	{
		ID:          StaffMarkerDisciplinary,
		Name:        "Under Disciplinary",
		Description: "Marks the holder as being under a disciplinary, such as a hiatus. Carries no power on its own, and is what a disciplinary that strips everything else leaves behind.",
		Category:    "Markers",
		Legacy:      []string{"dt.marker"},
	},
})
