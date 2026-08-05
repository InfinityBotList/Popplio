package bot

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"popplio/config"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
)

func cmdStats() *Command {
	return &Command{
		Name:        "stats",
		Description: "Bot statistics",
		Run: func(c *Ctx) error {
			var (
				revision = "unknown"
				modified = "unknown"
				version  = "unknown"
			)

			if bi, ok := debug.ReadBuildInfo(); ok {
				version = bi.Main.Version

				for _, setting := range bi.Settings {
					switch setting.Key {
					case "vcs.revision":
						revision = setting.Value
					case "vcs.modified":
						modified = setting.Value
					}
				}
			}

			return c.Send(discord.MessageCreate{
				Embeds: []discord.Embed{{
					Title: "Infernoplex Statistics",
					Fields: []discord.EmbedField{
						{Name: "Bot Version:", Value: version, Inline: validators.TruePtr},
						{Name: "Go Version:", Value: runtime.Version(), Inline: validators.TruePtr},
						{Name: "Git Commit:", Value: revision, Inline: validators.TruePtr},
						{Name: "Modified:", Value: modified, Inline: validators.TruePtr},
						{Name: "Built On:", Value: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), Inline: validators.TruePtr},
						{Name: "Current Environment:", Value: config.CurrentEnv, Inline: validators.TruePtr},
					},
				}},
			})
		},
	}
}
