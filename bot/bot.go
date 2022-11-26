package bot

import (
	"popplio/bot/commands/staff"
)

func LoadBot() {
	staff.Register()

	//botapi.RegisterWithAPI(state.Discord)

	//botapi.Start(state.Discord)
}
