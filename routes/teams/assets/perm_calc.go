package assets

import (
	"errors"
	"popplio/teams"
)

// Checks that the perms selected can be added with the perms the manager has
func CheckPerms(managerPerms []teams.TeamPermission, newPerms []teams.TeamPermission) ([]teams.TeamPermission, error) {
	var userPerms = teams.NewPermissionManager(newPerms).Perms()
	var managerPermsManager = teams.NewPermissionManager(managerPerms)

	if managerPermsManager.Has(teams.TeamPermissionOwner) {
		return userPerms, nil
	}

	if !managerPermsManager.Has(teams.TeamPermissionManageTeamMembers) {
		return nil, errors.New("you can't manage team members")
	}

	// Ensure that all perms in userPerms are in managerPerms since not owner
	for _, perm := range userPerms {
		if !managerPermsManager.Has(perm) {
			return nil, errors.New("you can't grant a permission you don't have")
		}
	}

	return userPerms, nil
}
