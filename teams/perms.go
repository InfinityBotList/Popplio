package teams

import "golang.org/x/exp/slices"

type TeamPermission string // TeamPermission is a permission that a team can have

/*
TODO:

- Add more permissions
- Arcadia task to ensure all owners have a team_member entry with the OWNER permission
*/

const (
	TeamPermissionUndefined                 TeamPermission = ""
	TeamPermissionEditBotSettings           TeamPermission = "EDIT_BOT_SETTINGS"
	TeamPermissionAddNewBots                TeamPermission = "ADD_NEW_BOTS"
	TeamPermissionResubmitBots              TeamPermission = "RESUBMIT_BOTS"
	TeamPermissionCertifyBots               TeamPermission = "CERTIFY_BOTS"
	TeamPermissionResetBotTokens            TeamPermission = "RESET_BOT_TOKEN"
	TeamPermissionEditBotWebhooks           TeamPermission = "EDIT_BOT_WEBHOOKS"
	TeamPermissionTestBotWebhooks           TeamPermission = "TEST_BOT_WEBHOOKS"
	TeamPermissionSetBotVanity              TeamPermission = "SET_BOT_VANITY"
	TeamPermissionEditTeamInfo              TeamPermission = "EDIT_TEAM_INFO"
	TeamPermissionAddTeamMembers            TeamPermission = "ADD_TEAM_MEMBERS"
	TeamPermissionRemoveTeamMembers         TeamPermission = "REMOVE_TEAM_MEMBERS"
	TeamPermissionEditTeamMemberPermissions TeamPermission = "EDIT_TEAM_MEMBER_PERMISSIONS"
	TeamPermissionDeleteBots                TeamPermission = "DELETE_BOTS"
	TeamPermissionOwner                     TeamPermission = "OWNER"
)

type PermDetailMap struct {
	ID   TeamPermission `json:"id"`
	Name string         `json:"name"`
	Desc string         `json:"desc"`
}

var TeamPermDetails = []PermDetailMap{
	{TeamPermissionUndefined, "Undefined", "Undefined"},
	{TeamPermissionEditBotSettings, "Edit Bot Settings", "Edit bot settings for bots on the team"},
	{TeamPermissionAddNewBots, "Add New Bots", "Add new bots to the team or allow transferring bots to this team"},
	{TeamPermissionResubmitBots, "Resubmit Bots", "Resubmit bots on the team"},
	{TeamPermissionCertifyBots, "Certify Bots", "Request certification for bots on the team"},
	{TeamPermissionResetBotTokens, "Reset Bot Tokens", "Reset the API token of bots on the team"},
	{TeamPermissionEditBotWebhooks, "Edit Bot Webhooks", "Edit bot webhook settings. Note that 'Test Bot Webhooks' is a separate permission and is required to test webhooks."},
	{TeamPermissionTestBotWebhooks, "Test Bot Webhooks", "Test bot webhooks. Note that this is a separate permission from 'Edit Bot Webhooks' and is required to test webhooks."},
	{TeamPermissionSetBotVanity, "Set Bot Vanity", "Set vanity URLs for bots on the team"},
	{TeamPermissionEditTeamInfo, "Edit Team Information", "Edit the team's name and avatar"},
	{TeamPermissionAddTeamMembers, "Add Team Members", "Add team members to the team. Also needs 'Edit Team Member Permissions'"},
	{TeamPermissionRemoveTeamMembers, "Remove Team Members", "Remove team members from the team if they have all the permissions of the user they are removing. Does **NOT** need 'Edit Team Member Permissions'"},
	{TeamPermissionEditTeamMemberPermissions, "Edit Team Member Permissions", "Edit team members' permissions"},
	{TeamPermissionDeleteBots, "Delete Bots", "Delete bots from the team. This is a very dangerous permission and should usually never be given to anyone."},
	{TeamPermissionOwner, "Owner", "Do everything (as they're owner). This is a very dangerous permission and should usually never be given to anyone."},
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
