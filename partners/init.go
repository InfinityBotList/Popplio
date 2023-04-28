package partners

import (
	"popplio/state"
	"popplio/types"

	"github.com/infinitybotlist/eureka/dovewing"
)

func processPartner(partner *types.Partner) *types.Partner {
	if partner.User != nil {
		return partner
	}

	u, err := dovewing.GetDiscordUser(state.Context, partner.UserID)

	if err != nil {
		panic("Error getting discord user: " + err.Error())
	}

	partner.User = u

	return partner
}

func Setup() {
	err := state.Validator.Struct(Partners)

	if err != nil {
		panic("Partner validation failed: " + err.Error())
	}

	for i := 0; i < len(Partners.Featured); i++ {
		Partners.Featured[i] = processPartner(Partners.Featured[i])
	}

	for i := 0; i < len(Partners.BotPartners); i++ {
		Partners.BotPartners[i] = processPartner(Partners.BotPartners[i])
	}

	for i := 0; i < len(Partners.BotListPartners); i++ {
		Partners.BotListPartners[i] = processPartner(Partners.BotListPartners[i])
	}
}
