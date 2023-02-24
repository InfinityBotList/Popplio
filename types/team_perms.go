package types

import "golang.org/x/exp/slices"

type TeamPermission string // TeamPermission is a permission that a team can have

/*
TODO:

- Add more permissions
- Arcadia task to ensure all owners have a team_member entry with the OWNER permission
*/

const (
	TeamPermissionEditBotSettings   TeamPermission = "EDIT_BOT_SETTINGS"   // TeamPermissionManageBots is the permission to edit bot settings
	TeamPermissionAddNewBots        TeamPermission = "ADD_NEW_BOTS"        // TeamPermissionAddNewBots is the permission to add new bots to the team
	TeamPermissionDeleteBots        TeamPermission = "DELETE_BOTS"         // TeamPermissionDeleteBots is the permission to delete bots from the team
	TeamPermissionResetBotToken     TeamPermission = "RESET_BOT_TOKEN"     // TeamPermissionResetToken is the permission to reset the bot's API token
	TeamPermissionEditBotWebhooks   TeamPermission = "EDIT_BOT_WEBHOOKS"   // TeamPermissionEditBotWebhooks is the permission to edit bot webhook settings
	TeamPermissionSetBotVanity      TeamPermission = "SET_BOT_VANITY"      // TeamPermissionSetVanity is the permission to edit vanities for bots
	TeamPermissionManageTeam        TeamPermission = "MANAGE_TEAM"         // TeamPermissionManageTeam is the permission to edit team settings
	TeamPermissionManageTeamMembers TeamPermission = "MANAGE_TEAM_MEMBERS" // TeamPermissionAddNewTeamMembers is the permission to add or remove team members from the team
	TeamPermissionOwner             TeamPermission = "OWNER"               // TeamPermissionOwner is the permission to do everything (as they're owner)
)

// PermissionManager is a manager for team permissions
type PermissionManager struct {
	perms []TeamPermission
}

func NewPermissionManager(perms []TeamPermission) *PermissionManager {
	return &PermissionManager{perms: perms}
}

// Has returns whether the team member has the specified permission
func (m *PermissionManager) Has(perm TeamPermission) bool {
	return slices.Contains(m.perms, perm) || slices.Contains(m.perms, TeamPermissionOwner)
}
