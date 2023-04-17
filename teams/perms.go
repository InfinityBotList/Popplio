package teams

import "golang.org/x/exp/slices"

type TeamPermission string // TeamPermission is a permission that a team can have

/*
TODO:

- Add more permissions
- Arcadia task to ensure all owners have a team_member entry with the OWNER permission
*/

const (
	TeamPermissionUndefined TeamPermission = ""

	// Bot permissions
	TeamPermissionEditBotSettings TeamPermission = "EDIT_BOT_SETTINGS"
	TeamPermissionAddNewBots      TeamPermission = "ADD_NEW_BOTS"
	TeamPermissionResubmitBots    TeamPermission = "RESUBMIT_BOTS"
	TeamPermissionCertifyBots     TeamPermission = "CERTIFY_BOTS"
	TeamPermissionResetBotTokens  TeamPermission = "RESET_BOT_TOKEN"
	TeamPermissionEditBotWebhooks TeamPermission = "EDIT_BOT_WEBHOOKS"
	TeamPermissionTestBotWebhooks TeamPermission = "TEST_BOT_WEBHOOKS"
	TeamPermissionSetBotVanity    TeamPermission = "SET_BOT_VANITY"
	TeamPermissionDeleteBots      TeamPermission = "DELETE_BOTS"

	// Server permissions
	TeamPermissionEditServerSettings TeamPermission = "EDIT_SERVER_SETTINGS"
	TeamPermissionAddNewServers      TeamPermission = "ADD_NEW_SERVERS"
	TeamPermissionCertifyServers     TeamPermission = "CERTIFY_SERVERS"
	TeamPermissionResetServerTokens  TeamPermission = "RESET_SERVER_TOKEN"
	TeamPermissionEditServerWebhooks TeamPermission = "EDIT_SERVER_WEBHOOKS"
	TeamPermissionTestServerWebhooks TeamPermission = "TEST_SERVER_WEBHOOKS"
	TeamPermissionSetServerVanity    TeamPermission = "SET_SERVER_VANITY"
	TeamPermissionDeleteServers      TeamPermission = "DELETE_SERVERS"

	// Common permissions
	TeamPermissionEditTeamInfo              TeamPermission = "EDIT_TEAM_INFO"
	TeamPermissionAddTeamMembers            TeamPermission = "ADD_TEAM_MEMBERS"
	TeamPermissionRemoveTeamMembers         TeamPermission = "REMOVE_TEAM_MEMBERS"
	TeamPermissionEditTeamMemberPermissions TeamPermission = "EDIT_TEAM_MEMBER_PERMISSIONS"

	// Owner permission
	TeamPermissionOwner TeamPermission = "OWNER"
)

type PermDetailMap struct {
	ID    TeamPermission `json:"id"`
	Name  string         `json:"name"`
	Desc  string         `json:"desc"`
	Group string         `json:"group"`
}

var TeamPermDetails = []PermDetailMap{
	{TeamPermissionUndefined, "Undefined", "Undefined", "undefined"},

	// Bot permissions
	{TeamPermissionEditBotSettings, "Edit Bot Settings", "Edit bot settings for bots on the team", "Bot"},
	{TeamPermissionAddNewBots, "Add New Bots", "Add new bots to the team or allow transferring bots to this team", "Bot"},
	{TeamPermissionResubmitBots, "Resubmit Bots", "Resubmit bots on the team", "Bot"},
	{TeamPermissionCertifyBots, "Certify Bots", "Request certification for bots on the team", "Bot"},
	{TeamPermissionResetBotTokens, "Reset Bot Tokens", "Reset the API token of bots on the team", "Bot"},
	{TeamPermissionEditBotWebhooks, "Edit Bot Webhooks", "Edit bot webhook settings. Note that 'Test Bot Webhooks' is a separate permission and is required to test webhooks.", "Bot"},
	{TeamPermissionTestBotWebhooks, "Test Bot Webhooks", "Test bot webhooks. Note that this is a separate permission from 'Edit Bot Webhooks' and is required to test webhooks.", "Bot"},
	{TeamPermissionSetBotVanity, "Set Bot Vanity", "Set vanity URLs for bots on the team", "Bot"},
	{TeamPermissionDeleteBots, "Delete Bots", "Delete bots from the team. This is a very dangerous permission and should usually never be given to anyone.", "Bot"},

	// Server permissions
	{TeamPermissionEditServerSettings, "Edit Server Settings", "Edit server settings for servers on the team", "Server"},
	{TeamPermissionAddNewServers, "Add New Servers", "Add new servers to the team or allow transferring servers to this team", "Server"},
	{TeamPermissionCertifyServers, "Certify Servers", "Request certification for servers on the team", "Server"},
	{TeamPermissionResetServerTokens, "Reset Server Tokens", "Reset the API token of servers on the team", "Server"},
	{TeamPermissionEditServerWebhooks, "Edit Server Webhooks", "Edit server webhook settings. Note that 'Test Server Webhooks' is a separate permission and is required to test webhooks.", "Server"},
	{TeamPermissionTestServerWebhooks, "Test Server Webhooks", "Test server webhooks. Note that this is a separate permission from 'Edit Server Webhooks' and is required to test webhooks.", "Server"},
	{TeamPermissionSetServerVanity, "Set Server Vanity", "Set vanity URLs for servers on the team", "Server"},
	{TeamPermissionDeleteServers, "Delete Servers", "Delete servers from the team. This is a very dangerous permission and should usually never be given to anyone.", "Server"},

	// Team permissions
	{TeamPermissionEditTeamInfo, "Edit Team Information", "Edit the team's name and avatar", "Team"},
	{TeamPermissionAddTeamMembers, "Add Team Members", "Add team members to the team. Also needs 'Edit Team Member Permissions'", "Team"},
	{TeamPermissionRemoveTeamMembers, "Remove Team Members", "Remove team members from the team if they have all the permissions of the user they are removing. Does **NOT** need 'Edit Team Member Permissions'", "Team"},
	{TeamPermissionEditTeamMemberPermissions, "Edit Team Member Permissions", "Edit team members' permissions", "Team"},

	// Common permission
	{TeamPermissionOwner, "Owner", "Do everything (as they're owner). This is a very dangerous permission and should usually never be given to anyone.", "Common"},
}

func isValidPerm(perm TeamPermission) bool {
	for _, p := range TeamPermDetails {
		if p.ID == perm {
			return true
		}
	}

	return false
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
		if perm == TeamPermissionUndefined || !isValidPerm(perm) {
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
