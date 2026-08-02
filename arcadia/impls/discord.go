package impls

import (
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// Colours used by the mod-log embeds. BLURPLE is serenity's Colour::BLURPLE,
// which the Claim embed uses.
const (
	ColourBlurple = 0x7289DA
	ColourGreen   = 0x00ff00
	ColourRed     = 0xFF0000
	// ColourRedLower is the lowercase 0xff0000 literal the certify embeds use.
	// Identical value, kept separate only so the port reads like the source.
	ColourRedLower = 0xff0000
)

var (
	truePtr  = boolPtr(true)
	falsePtr = boolPtr(false)
)

func boolPtr(b bool) *bool { return &b }

// InlineTrue and InlineFalse are the embed field inline flags.
func InlineTrue() *bool  { return truePtr }
func InlineFalse() *bool { return falsePtr }

// MemberOnGuild reports whether a user is a cached member of a guild. A guild
// that is not cached reads as "not a member", as upstream does.
func MemberOnGuild(guildID snowflake.ID, userID snowflake.ID) bool {
	_, ok := state.Discord.Caches().Member(guildID, userID)
	return ok
}

// SendModLog posts a message to the configured mod-logs channel.
func SendModLog(msg discord.MessageCreate) error {
	_, err := state.Discord.Rest().CreateMessage(state.Config.Channels.ModLogs, msg)
	return err
}

// SendChannel posts a message to an arbitrary channel.
func SendChannel(channelID snowflake.ID, msg discord.MessageCreate) error {
	_, err := state.Discord.Rest().CreateMessage(channelID, msg)
	return err
}

// AddRole grants a role, recording an audit-log reason.
func AddRole(guildID, userID, roleID snowflake.ID, reason string) error {
	return state.Discord.Rest().AddMemberRole(guildID, userID, roleID, rest.WithReason(reason))
}

// RemoveRole revokes a role, recording an audit-log reason.
func RemoveRole(guildID, userID, roleID snowflake.ID, reason string) error {
	return state.Discord.Rest().RemoveMemberRole(guildID, userID, roleID, rest.WithReason(reason))
}

// KickMember removes a member from a guild, recording an audit-log reason.
func KickMember(guildID, userID snowflake.ID, reason string) error {
	return state.Discord.Rest().RemoveMember(guildID, userID, rest.WithReason(reason))
}

// Footer builds an embed footer.
func Footer(text string) *discord.EmbedFooter {
	return &discord.EmbedFooter{Text: text}
}
