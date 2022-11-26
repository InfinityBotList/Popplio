// Defines staff commands
package staff

import "popplio/bot/botapi"

func Register() {
	botapi.AddCommand(botapi.Command{
		Name:        "staff",
		Description: "Staff commands",
		Callback:    func(*botapi.Context) {},
		Subcommands: []botapi.Subcommand{
			{
				Name:        "testcmd",
				Description: "Test command",
				Arguments: []botapi.Value{
					{
						Name:        "arg1",
						Description: "Argument 1",
						Type:        botapi.ArgumentTypeString,
						Required:    true,
					},
				},
				Callback: func(ctx *botapi.Context) {
					ctx.Reply("Test command"+ctx.Values.Get("arg1").String(), false)
				},
			},
		},
	})
}
