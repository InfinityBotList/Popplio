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

	u, err := dovewing.GetUser(state.Context, partner.UserID, state.DovewingPlatformDiscord)

	if err != nil {
		panic("Error getting discord user: " + err.Error())
	}

	partner.User = u

	return partner
}

func Setup() {
	setupPartnerList()

	for p := range Partners.BotListPartners {
		_, err := state.Pool.Exec(state.Context, "INSERT INTO partners (id, name, short, user_id, image, links, type) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET name = $2, short = $3, user_id = $4, image = $5, links = $6, type = $7", Partners.BotListPartners[p].ID, Partners.BotListPartners[p].Name, Partners.BotListPartners[p].Short, Partners.BotListPartners[p].UserID, Partners.BotListPartners[p].Image, Partners.BotListPartners[p].Links, "botlist")

		if err != nil {
			panic("Error inserting botlist partner: " + err.Error())
		}
	}

	for p := range Partners.BotPartners {
		_, err := state.Pool.Exec(state.Context, "INSERT INTO partners (id, name, short, user_id, image, links, type) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET name = $2, short = $3, user_id = $4, image = $5, links = $6, type = $7", Partners.BotPartners[p].ID, Partners.BotPartners[p].Name, Partners.BotPartners[p].Short, Partners.BotPartners[p].UserID, Partners.BotPartners[p].Image, Partners.BotPartners[p].Links, "bot")

		if err != nil {
			panic("Error inserting bot partner: " + err.Error())
		}
	}

	for p := range Partners.Featured {
		_, err := state.Pool.Exec(state.Context, "INSERT INTO partners (id, name, short, user_id, image, links, type) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO UPDATE SET name = $2, short = $3, user_id = $4, image = $5, links = $6, type = $7", Partners.Featured[p].ID, Partners.Featured[p].Name, Partners.Featured[p].Short, Partners.Featured[p].UserID, Partners.Featured[p].Image, Partners.Featured[p].Links, "featured")

		if err != nil {
			panic("Error inserting featured partner: " + err.Error() + " " + Partners.Featured[p].ID)
		}
	}

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
