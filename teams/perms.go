package teams

import "golang.org/x/exp/slices"

type TeamPermission string // TeamPermission is a permission that a team can have

/*
TODO:

- Add more permissions
- Arcadia task to ensure all owners have a team_member entry with the OWNER permission
*/

// MAKE SURE TO UPDATE INFINITY-NEXT utils/teams/teamPerms.ts WHEN UPDATING THIS
const (
	TeamPermissionUndefined         TeamPermission = ""                    // TeamPermissionUndefined is the default permission
	TeamPermissionEditBotSettings   TeamPermission = "EDIT_BOT_SETTINGS"   // TeamPermissionManageBots is the permission to edit bot settings
	TeamPermissionAddNewBots        TeamPermission = "ADD_NEW_BOTS"        // TeamPermissionAddNewBots is the permission to add new bots to the team
	TeamPermissionResubmitBots      TeamPermission = "RESUBMIT_BOTS"       // TeamPermissionResubmitBots is the permission to resubmit bots on the team
	TeamPermissionCertifyBots       TeamPermission = "CERTIFY_BOTS"        // TeamPermissionCertifyBots is the permission to request certification for bots on the team
	TeamPermissionDeleteBots        TeamPermission = "DELETE_BOTS"         // TeamPermissionDeleteBots is the permission to delete bots from the team
	TeamPermissionResetBotTokens    TeamPermission = "RESET_BOT_TOKEN"     // TeamPermissionResetToken is the permission to reset the bot's API token
	TeamPermissionEditBotWebhooks   TeamPermission = "EDIT_BOT_WEBHOOKS"   // TeamPermissionEditBotWebhooks is the permission to edit bot webhook settings
	TeamPermissionTestBotWebhooks   TeamPermission = "TEST_BOT_WEBHOOKS"   // TeamPermissionTestBotWebhooks is the permission to test bot webhooks
	TeamPermissionSetBotVanity      TeamPermission = "SET_BOT_VANITY"      // TeamPermissionSetVanity is the permission to edit vanities for bots
	TeamPermissionManageTeam        TeamPermission = "MANAGE_TEAM"         // TeamPermissionManageTeam is the permission to edit team settings
	TeamPermissionManageTeamMembers TeamPermission = "MANAGE_TEAM_MEMBERS" // TeamPermissionAddNewTeamMembers is the permission to add or remove team members from the team
	TeamPermissionOwner             TeamPermission = "OWNER"               // TeamPermissionOwner is the permission to do everything (as they're owner)
)

type PermDetailMap struct {
	ID   TeamPermission `json:"id"`
	Name string         `json:"name"`
	Desc string         `json:"desc"`
}

var TeamPermDetails = []PermDetailMap{
	{TeamPermissionUndefined, "Undefined", "Undefined"},
	{TeamPermissionEditBotSettings, "Edit Bot Settings", "Edit bot settings for bots on the team"},
	{TeamPermissionAddNewBots, "Add New Bots", "Add new bots to the team"},
	{TeamPermissionResubmitBots, "Resubmit Bots", "Resubmit bots on the team"},
	{TeamPermissionCertifyBots, "Certify Bots", "Request certification for bots on the team"},
	{TeamPermissionDeleteBots, "Delete Bots", "Delete bots from the team"},
	{TeamPermissionResetBotTokens, "Reset Bot Tokens", "Reset the API token of bots on the team"},
	{TeamPermissionEditBotWebhooks, "Edit Bot Webhooks", "Edit bot webhook settings. Note that 'Test Bot Webhooks' is a separate permission and is required to test webhooks."},
	{TeamPermissionTestBotWebhooks, "Test Bot Webhooks", "Test bot webhooks. Note that this is a separate permission from 'Edit Bot Webhooks' and is required to test webhooks."},
	{TeamPermissionSetBotVanity, "Set Bot Vanity", "Set vanity URLs for bots on the team"},
	{TeamPermissionManageTeam, "Manage Team", "Edit team settings"},
	{TeamPermissionManageTeamMembers, "Manage Team Members", "Add or remove team members from the team as well as edit their permissions"},
	{TeamPermissionOwner, "Owner", "Do everything (as they're owner)"},
}

// PermissionManager is a manager for team permissions
type PermissionManager struct {
	perms []TeamPermission
}

// NewPermissionManager creates a new permission manager from a list of permissions
// It will remove duplicates and undefined permissions
func NewPermissionManager(perms []TeamPermission) *PermissionManager {
	var uniquePerms []TeamPermission

	for _, perm := range perms {
		if perm == TeamPermissionUndefined {
			continue
		}

		if !slices.Contains(uniquePerms, perm) {
			uniquePerms = append(uniquePerms, perm)
		}
	}

	return &PermissionManager{perms: uniquePerms}
}

// Has returns whether the team member has the specified permission
func (m *PermissionManager) Has(perm TeamPermission) bool {
	return slices.Contains(m.perms, perm) || slices.Contains(m.perms, TeamPermissionOwner)
}

// Perms returns the list of permissions the team member has
func (m *PermissionManager) Perms() []TeamPermission {
	return m.perms
}

func (m *PermissionManager) HasSomePerms() bool {
	return len(m.perms) > 0
}
