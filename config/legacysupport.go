package config

var legacyWebhooks = [...]string{
	"827234880122650654",  // Succubus, marshall
	"845214061511180298",  // Wyvern, Ben/Connor
	"587152333519978559",  // Primo, compiles (Dan), ashmw (AshMW)
	"892836355074306058",  // GamingBuddy, muckmuck96
	"924540290461736971",  // Watcher, frostlord_
	"1000125868938633297", // DittoBOT, .skylarr.
	"187636089073172481",  // DuckHunt, canarduck (Canarde/EyesOfCreeper)
}

func UseLegacyWebhooks(botId string) bool {
	for _, id := range legacyWebhooks {
		if id == botId {
			return true
		}
	}

	return false
}
