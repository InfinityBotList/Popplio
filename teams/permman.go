package teams

import (
	"popplio/types"

	"golang.org/x/exp/slices"
)

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
