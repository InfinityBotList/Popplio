package bot

import (
	"strings"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"go.uber.org/zap"
)

// Small shared pieces of the editor: reading a stored permission list exactly,
// rendering one inline, and answering an interaction privately.

// heldMap reads a stored permission list exactly, without the super permission's
// implication: an editor must show what would be written back, not what the
// holder can do.
func heldMap(list []perms.Perm) map[perms.Perm]bool {
	out := make(map[perms.Perm]bool, len(list))

	for _, p := range list {
		out[p] = true
	}

	return out
}

func inlineList(list []perms.Perm) string {
	out := make([]string, 0, len(list))

	for _, p := range list {
		out = append(out, "``"+string(p)+"``")
	}

	return strings.Join(out, ", ")
}

func selectValues(e *events.ComponentInteractionCreate) []string {
	data, ok := e.Data.(discord.StringSelectMenuInteractionData)

	if !ok {
		return nil
	}

	return data.Values
}

// respondEphemeral answers an interaction with a message only its clicker sees,
// which is how a refusal reaches them without editing the shared message.
func respondEphemeral(e *events.ComponentInteractionCreate, content string) {
	err := e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{{
			Description: content,
			Color:       impls.ColourRed,
		}},
		Flags: discord.MessageFlagEphemeral,
	})

	if err != nil {
		state.Logger.Error("Failed to answer a permission editor interaction", zap.Error(err))
	}
}
