package bot

import (
	"fmt"
	"strings"

	"popplio/arcadia/impls"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
)

// How permissions are rendered into Discord, shared by the commands and the
// interactive editor.
//
// The shapes here exist because a 35-permission set is unreadable as a flat
// list and a full catalogue listing does not fit in one Discord message.

// permissionFields renders a permission set as one embed field per category,
// which is the only way a 30-permission set stays readable.
func permissionFields(set perms.Set) []discord.EmbedField {
	if set.IsEmpty() {
		return []discord.EmbedField{{Name: "Permissions", Value: "None", Inline: impls.InlineFalse()}}
	}

	if set.IsSuper() {
		return []discord.EmbedField{{
			Name:   "Permissions",
			Value:  "**Administrator** — everything",
			Inline: impls.InlineFalse(),
		}}
	}

	var fields []discord.EmbedField

	for _, category := range perms.Staff.Categories() {
		var held []perms.Perm

		for _, d := range perms.Staff.InCategory(category) {
			if set.Has(d.ID) {
				held = append(held, d.ID)
			}
		}

		if len(held) == 0 {
			continue
		}

		fields = append(fields, discord.EmbedField{
			Name:   category,
			Value:  codeList(held),
			Inline: impls.InlineTrue(),
		})
	}

	// Permissions owned by another service still belong to the holder and are
	// shown rather than quietly hidden.
	if undeclared := set.Undeclared(); len(undeclared) > 0 {
		fields = append(fields, discord.EmbedField{
			Name:   "Other services",
			Value:  codeList(undeclared),
			Inline: impls.InlineFalse(),
		})
	}

	return fields
}

func codeList(list []perms.Perm) string {
	out := make([]string, 0, len(list))

	for _, p := range list {
		out = append(out, "``"+string(p)+"``")
	}

	return strings.Join(out, "\n")
}

// chunkMessage splits text on line boundaries to fit Discord's message limit,
// preserving the text exactly across the chunks.
func chunkMessage(s string, limit int) []string {
	var (
		out     []string
		current []string
		size    int
	)

	for _, line := range strings.Split(s, "\n") {
		if size+len(line)+1 > limit && len(current) > 0 {
			out = append(out, strings.Join(current, "\n"))
			current, size = nil, 0
		}

		current = append(current, line)
		size += len(line) + 1
	}

	if len(current) > 0 {
		out = append(out, strings.Join(current, "\n"))
	}

	return out
}

// logRoleChange writes to the staff log channel, the same place the resync task
// reports to. A permission change nobody can see afterwards is not accountable.
func logRoleChange(c *Ctx, title, description string) {
	err := impls.SendChannel(state.Config.Channels.StaffLogs, discord.MessageCreate{
		Embeds: []discord.Embed{{
			Title:       title,
			Description: description,
			Fields: []discord.EmbedField{
				{Name: "By", Value: fmt.Sprintf("<@%s>", c.Author.ID), Inline: impls.InlineTrue()},
			},
			Color: impls.ColourGreen,
		}},
	})

	if err != nil {
		state.Logger.Error("Failed to write staff log for a permission change: " + err.Error())
	}
}
