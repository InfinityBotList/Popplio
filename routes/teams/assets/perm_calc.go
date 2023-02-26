package assets

import (
	"errors"
	"popplio/teams"
)

// Returns the unique elements between two arrays
func diffArray(a []teams.TeamPermission, b []teams.TeamPermission) []teams.TeamPermission {
	var diff = []teams.TeamPermission{}

	// Get unique elements in a that are not in b
	for _, aVal := range a {
		var found bool = false
		for _, bVal := range b {
			if aVal == bVal {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, aVal)
		}
	}

	// Get unique elements in b that are not in a
	for _, bVal := range b {
		var found bool = false
		for _, aVal := range a {
			if bVal == aVal {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, bVal)
		}
	}

	return diff
}

// Checks that the perms selected can be added with the perms the manager has
func CheckPerms(managerPerms []teams.TeamPermission, oldPerms []teams.TeamPermission, newPerms []teams.TeamPermission) ([]teams.TeamPermission, error) {
	var userPerms = teams.NewPermissionManager(newPerms).Perms()
	var managerPermsManager = teams.NewPermissionManager(managerPerms)

	if managerPermsManager.Has(teams.TeamPermissionOwner) {
		return userPerms, nil
	}

	if !managerPermsManager.Has(teams.TeamPermissionEditTeamMemberPermissions) {
		return nil, errors.New("you can't manage team members")
	}

	// Get changes between old and new perms
	diffPerms := diffArray(oldPerms, newPerms)

	// Ensure that all perms in userPerms are in managerPerms since not owner
	for _, perm := range diffPerms {
		if !managerPermsManager.Has(perm) {
			return nil, errors.New("you can't grant a permission you don't have")
		}
	}

	return userPerms, nil
}
