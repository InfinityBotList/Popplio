package assets

import (
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/snowflake/v2"
	"golang.org/x/exp/slices"
)

func CheckUserBoosterStatus(id snowflake.ID) types.BoosterStatus {
	// Check member is a booster
	m, ok := state.Discord.Caches().Member(state.Config.Servers.Main, id)

	if !ok {
		return types.BoosterStatus{
			Remark:    "Member not found on server",
			IsBooster: false,
		}
	}

	// Check if member has booster role
	roles := state.Config.Roles.PremiumRoles.Parse()
	for _, role := range m.RoleIDs {
		if slices.Contains(roles, role) {
			// Member has booster role
			return types.BoosterStatus{
				IsBooster: true,
			}
		}
	}

	return types.BoosterStatus{
		Remark:    "Member does not have booster role",
		IsBooster: false,
	}
}
