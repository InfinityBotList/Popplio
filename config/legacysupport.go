package config

var legacyWebhooks = [...]string{
	"580159619415146506",  // Emoji Generator, builderb
	"371789181954818050",  // Hangman, builderb
	"325892959130222592",  // Starboat, builderb
	"906732932918542366",  // Minotaur, builderb
	"434556304661544960",  // Waifu, builderb
	"827234880122650654",  // Succubus, marshall
	"845214061511180298",  // Wyvern, Ben/Connor
	"1068627569760550994", // Threaded, thatbadname
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
