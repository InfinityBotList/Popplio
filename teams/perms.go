package teams

import (
	"popplio/types"

	"golang.org/x/exp/slices"
)

/*
TODO:

- Add more permissions
- Arcadia task to ensure all owners have a team_member entry with the OWNER permission
*/

const (
	TeamPermissionUndefined types.TeamPermission = ""

	// Bot permissions
	TeamPermissionEditBotSettings       types.TeamPermission = "EDIT_BOT_SETTINGS"
	TeamPermissionAddNewBots            types.TeamPermission = "ADD_NEW_BOTS"
	TeamPermissionResubmitBots          types.TeamPermission = "RESUBMIT_BOTS"
	TeamPermissionCertifyBots           types.TeamPermission = "CERTIFY_BOTS"
	TeamPermissionViewExistingBotTokens types.TeamPermission = "VIEW_EXISTING_BOT_TOKENS"
	TeamPermissionResetBotTokens        types.TeamPermission = "RESET_BOT_TOKEN"
	TeamPermissionEditBotWebhooks       types.TeamPermission = "EDIT_BOT_WEBHOOKS"
	TeamPermissionTestBotWebhooks       types.TeamPermission = "TEST_BOT_WEBHOOKS"
	TeamPermissionGetBotWebhookLogs     types.TeamPermission = "GET_BOT_WEBHOOK_LOGS"
	TeamPermissionRetryBotWebhookLogs   types.TeamPermission = "RETRY_BOT_WEBHOOK_LOGS"
	TeamPermissionDeleteBotWebhookLogs  types.TeamPermission = "DELETE_BOT_WEBHOOK_LOGS"
	TeamPermissionSetBotVanity          types.TeamPermission = "SET_BOT_VANITY"
	TeamPermissionDeleteBots            types.TeamPermission = "DELETE_BOTS"

	// Server permissions
	TeamPermissionEditServerSettings       types.TeamPermission = "EDIT_SERVER_SETTINGS"
	TeamPermissionAddNewServers            types.TeamPermission = "ADD_NEW_SERVERS"
	TeamPermissionCertifyServers           types.TeamPermission = "CERTIFY_SERVERS"
	TeamPermissionViewExistingServerTokens types.TeamPermission = "VIEW_EXISTING_SERVER_TOKENS"
	TeamPermissionResetServerTokens        types.TeamPermission = "RESET_SERVER_TOKEN"
	TeamPermissionEditServerWebhooks       types.TeamPermission = "EDIT_SERVER_WEBHOOKS"
	TeamPermissionTestServerWebhooks       types.TeamPermission = "TEST_SERVER_WEBHOOKS"
	TeamPermissionGetServerWebhookLogs     types.TeamPermission = "GET_SERVER_WEBHOOK_LOGS"
	TeamPermissionRetryServerWebhookLogs   types.TeamPermission = "RETRY_SERVER_WEBHOOK_LOGS"
	TeamPermissionDeleteServerWebhookLogs  types.TeamPermission = "DELETE_SERVER_WEBHOOK_LOGS"
	TeamPermissionSetServerVanity          types.TeamPermission = "SET_SERVER_VANITY"
	TeamPermissionDeleteServers            types.TeamPermission = "DELETE_SERVERS"

	// Team permissions
	TeamPermissionEditTeamInfo              types.TeamPermission = "EDIT_TEAM_INFO"
	TeamPermissionAddTeamMembers            types.TeamPermission = "ADD_TEAM_MEMBERS"
	TeamPermissionRemoveTeamMembers         types.TeamPermission = "REMOVE_TEAM_MEMBERS"
	TeamPermissionEditTeamMemberPermissions types.TeamPermission = "EDIT_TEAM_MEMBER_PERMISSIONS"
	TeamPermissionEditTeamWebhooks          types.TeamPermission = "EDIT_TEAM_WEBHOOKS"
	TeamPermissionTestTeamWebhooks          types.TeamPermission = "TEST_TEAM_WEBHOOKS"
	TeamPermissionGetTeamWebhookLogs        types.TeamPermission = "GET_TEAM_WEBHOOK_LOGS"
	TeamPermissionRetryTeamWebhookLogs      types.TeamPermission = "RETRY_TEAM_WEBHOOK_LOGS"
	TeamPermissionDeleteTeamWebhookLogs     types.TeamPermission = "DELETE_TEAM_WEBHOOK_LOGS"

	// Owner permission
	TeamPermissionOwner types.TeamPermission = "OWNER"
)

var TeamPermDetails = []types.PermDetailMap{
	{ID: TeamPermissionUndefined, Name: "Undefined", Desc: "Undefined", Group: "undefined"},

	// Bot permissions
	{ID: TeamPermissionEditBotSettings, Name: "Edit Bot Settings", Desc: "Edit bot settings for bots on the team", Group: "Bot"},
	{ID: TeamPermissionAddNewBots, Name: "Add New Bots", Desc: "Add new bots to the team or allow transferring bots to this team", Group: "Bot"},
	{ID: TeamPermissionResubmitBots, Name: "Resubmit Bots", Desc: "Resubmit bots on the team", Group: "Bot"},
	{ID: TeamPermissionCertifyBots, Name: "Certify Bots", Desc: "Request certification for bots on the team", Group: "Bot"},
	{ID: TeamPermissionViewExistingBotTokens, Name: "View Existing Bot Tokens", Desc: "View existing API tokens of bots on the team. *DANGEROUS and a potential security risk as it can't even be audited*", Group: "Bot"},
	{ID: TeamPermissionResetBotTokens, Name: "Reset Bot Tokens", Desc: "Reset the API token of bots on the team. This is seperate from viewing existing bot tokens as that is a much greater security risk", Group: "Bot"},
	{ID: TeamPermissionEditBotWebhooks, Name: "Edit Bot Webhooks", Desc: "Edit bot webhook settings. Note that 'Test Bot Webhooks' is a separate permission and is required to test webhooks.", Group: "Bot"},
	{ID: TeamPermissionTestBotWebhooks, Name: "Test Bot Webhooks", Desc: "Test bot webhooks. Note that this is a separate permission from 'Edit Bot Webhooks' and is required to test webhooks.", Group: "Bot"},
	{ID: TeamPermissionGetBotWebhookLogs, Name: "Get Bot Webhook Logs", Desc: "Get bot webhook logs. Note that executing webhooks from webhook logs as well as deleting them are seperate permissions..", Group: "Bot"},
	{ID: TeamPermissionRetryBotWebhookLogs, Name: "Retry Bot Webhook Logs", Desc: "Retry execution of bot webhook logs. Usually requires 'Get Bot Webhook Logs' to be useful.", Group: "Bot"},
	{ID: TeamPermissionDeleteBotWebhookLogs, Name: "Delete Bot Webhook Logs", Desc: "Delete bot webhook logs. Usually requires 'Get Bot Webhook Logs' to be useful.", Group: "Bot"},
	{ID: TeamPermissionSetBotVanity, Name: "Set Bot Vanity", Desc: "Set vanity URLs for bots on the team", Group: "Bot"},
	{ID: TeamPermissionDeleteBots, Name: "Delete Bots", Desc: "Delete bots from the team. This is a very dangerous permission and should usually never be given to anyone.", Group: "Bot"},

	// Server permissions
	{ID: TeamPermissionEditServerSettings, Name: "Edit Server Settings", Desc: "Edit server settings for servers on the team", Group: "Server"},
	{ID: TeamPermissionAddNewServers, Name: "Add New Servers", Desc: "Add new servers to the team or allow transferring servers to this team", Group: "Server"},
	{ID: TeamPermissionCertifyServers, Name: "Certify Servers", Desc: "Request certification for servers on the team", Group: "Server"},
	{ID: TeamPermissionViewExistingServerTokens, Name: "View Existing Server Tokens", Desc: "View existing API tokens of servers on the team. *DANGEROUS and a potential security risk as it can't even be audited*", Group: "Bot"},
	{ID: TeamPermissionResetServerTokens, Name: "Reset Server Tokens", Desc: "Reset the API token of servers on the team", Group: "Server"},
	{ID: TeamPermissionEditServerWebhooks, Name: "Edit Server Webhooks", Desc: "Edit server webhook settings. Note that 'Test Server Webhooks' is a separate permission and is required to test webhooks.", Group: "Server"},
	{ID: TeamPermissionTestServerWebhooks, Name: "Test Server Webhooks", Desc: "Test server webhooks. Note that this is a separate permission from 'Edit Server Webhooks' and is required to test webhooks.", Group: "Server"},
	{ID: TeamPermissionGetServerWebhookLogs, Name: "Get Server Webhook Logs", Desc: "Get server webhook logs. Note that executing webhooks from webhook logs as well as deleting them are seperate permissions..", Group: "Server"},
	{ID: TeamPermissionRetryServerWebhookLogs, Name: "Retry Server Webhook Logs", Desc: "Retry execution of server webhook logs. Usually requires 'Get Server Webhook Logs' to be useful.", Group: "Server"},
	{ID: TeamPermissionDeleteServerWebhookLogs, Name: "Delete Server Webhook Logs", Desc: "Delete server webhook logs. Usually requires 'Get Server Webhook Logs' to be useful.", Group: "Server"},
	{ID: TeamPermissionSetServerVanity, Name: "Set Server Vanity", Desc: "Set vanity URLs for servers on the team", Group: "Server"},
	{ID: TeamPermissionDeleteServers, Name: "Delete Servers", Desc: "Delete servers from the team. This is a very dangerous permission and should usually never be given to anyone.", Group: "Server"},

	// Team permissions
	{ID: TeamPermissionEditTeamInfo, Name: "Edit Team Information", Desc: "Edit the team's name and avatar", Group: "Team"},
	{ID: TeamPermissionAddTeamMembers, Name: "Add Team Members", Desc: "Add team members to the team. Does **NOT** need 'Edit Team Member Permissions'", Group: "Team"},
	{ID: TeamPermissionRemoveTeamMembers, Name: "Remove Team Members", Desc: "Remove team members from the team if they have all the permissions of the user they are removing. Does **NOT** need 'Edit Team Member Permissions'", Group: "Team"},
	{ID: TeamPermissionEditTeamMemberPermissions, Name: "Edit Team Member Permissions", Desc: "Edit team members' permissions", Group: "Team"},
	{ID: TeamPermissionEditTeamWebhooks, Name: "Edit Team Webhooks", Desc: "Edit team webhook settings. Note that 'Test Team Webhooks' is a separate permission and is required to test webhooks.", Group: "Team"},
	{ID: TeamPermissionTestTeamWebhooks, Name: "Test Team Webhooks", Desc: "Test team webhooks. Note that this is a separate permission from 'Edit Test Webhooks' and is required to test webhooks.", Group: "Team"},
	{ID: TeamPermissionGetTeamWebhookLogs, Name: "Get Team Webhook Logs", Desc: "Get team webhook logs. Note that executing webhooks from webhook logs as well as deleting them are seperate permissions..", Group: "Team"},
	{ID: TeamPermissionRetryTeamWebhookLogs, Name: "Retry Team Webhook Logs", Desc: "Retry execution of team webhook logs. Usually requires 'Get Team Webhook Logs' to be useful.", Group: "Team"},
	{ID: TeamPermissionDeleteTeamWebhookLogs, Name: "Delete Team Webhook Logs", Desc: "Delete team webhook logs. Usually requires 'Get Team Webhook Logs' to be useful.", Group: "Team"},

	// Owner permission
	{ID: TeamPermissionOwner, Name: "Owner", Desc: "Do everything (as they're owner). This is a very dangerous permission and should usually never be given to anyone.", Group: "Common"},
}

func isValidPerm(perm types.TeamPermission) bool {
	for _, p := range TeamPermDetails {
		if p.ID == perm {
			return true
		}
	}

	return false
}

// PermissionManager is a manager for team permissions
type PermissionManager struct {
	perms []types.TeamPermission
}

// NewPermissionManager creates a new permission manager from a list of permissions
// It will remove duplicates and undefined permissions
func NewPermissionManager(perms []types.TeamPermission) *PermissionManager {
	var uniquePerms []types.TeamPermission

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
func (m *PermissionManager) Has(perm types.TeamPermission) bool {
	return slices.Contains(m.perms, perm) || slices.Contains(m.perms, TeamPermissionOwner)
}

// Perms returns the list of permissions the team member has
func (m *PermissionManager) Perms() []types.TeamPermission {
	return m.perms
}

func (m *PermissionManager) HasSomePerms() bool {
	return len(m.perms) > 0
}
