package assets

import (
	"popplio/state"
	"popplio/types"

	"golang.org/x/exp/slices"
)

func CheckUserBoosterStatus(id string) types.BoosterStatus {
	// Check member is a booster
	m, err := state.Discord.State.Member(state.Config.Servers.Main, id)

	if err != nil {
		return types.BoosterStatus{
			Remark:    "Member not found on server:" + err.Error(),
			IsBooster: false,
		}
	}

	// Check if member has booster role
	roles := state.Config.Roles.PremiumRoles.Parse()
	for _, role := range m.Roles {
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
