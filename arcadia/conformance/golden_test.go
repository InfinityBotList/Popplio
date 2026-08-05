// Package conformance holds the frozen-string tests.
//
// They assert on the source text of the other arcadia packages rather than on
// their behaviour, which keeps them in a package that pulls in no dependencies -
// and therefore runs even when a heavier package's test binary cannot be built.
package conformance

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestGoldenPanelStrings pins the panel's user-visible strings. They are FROZEN,
// typos included: "neeed", the differing capitalisation of "Entered"/"entered",
// the leading spaces and the parentheses on the bot_whitelist messages are all
// upstream.
//
// A deliberate change here belongs in arcadia/CONFORMANCE.md first.
func TestGoldenPanelStrings(t *testing.T) {
	golden := []string{
		// Auth. The frontend matches on these exact strings.
		"identityExpired",
		"sessionNotActive",
		"sessionAlreadyActive",
		"mfaNotSetup",
		"Invalid OTP Entered",
		"Invalid OTP entered",
		"This endpoint can only be used by pending and active sessions",
		"You are not a staff member",
		"You are not a staff member [not in db]",
		"You are not a staff member [no positions]",
		"Invalid version",
		"Invalid scope",
		"Invalid redirect url",

		// Partners.
		"Partner type does not exist",
		"Partner already exists",
		"Partner does not already exist",
		"Partner does not exist",
		"Links cannot be empty",
		"Link name cannot be empty",
		"Link URL cannot be empty",
		"Link URL must start with https://",
		"User does not exist",
		"You do not have permission to create partners [manage_partners]",
		"You do not have permission to update partners [manage_partners]",
		"You do not have permission to delete partners [manage_partners]",

		// Changelog is a hard stub.
		"You do not have permission to create changelog entries [not implemented]",

		// Blog.
		"Entry does not exist",
		"Entry with same id does not already exist",
		"You do not have permission to create blog entries [manage_blog]",
		"You do not have permission to update blog entries [manage_blog]",
		"You do not have permission to delete blog entries [manage_blog]",

		// Staff positions and members.
		"Positions have the same index",
		"Either 'a' or 'b' is lower than the lowest index of the member",
		"Index cannot be lower than 0",
		"Index to set is lower than or equal to the lowest index of the staff member",
		"Current index of position is lower than or equal to the lowest index of the staff member",
		"Index is lower than or equal to the lowest index of the staff member",
		"Index is lower than the lowest index of the member",
		"Role does not exist on the staff server",
		"Target has a lower index than the member",
		"You do not have permission to edit the following perms [neeed to delete position]: %s",
		"You do not have permission to edit the following perms: %s",
		"You do not have permission to swap indexes of staff positions [manage_staff_roles]",
		"You do not have permission to set the indexes of staff positions [manage_staff_roles]",
		"You do not have permission to create staff positions [manage_staff_roles]",
		"You do not have permission to edit staff positions [manage_staff_roles]",
		"You do not have permission to delete staff positions [manage_staff_roles]",
		"You do not have permission to edit staff members [manage_staff_members]",
		"You do not have permission to create staff disciplinary types [manage_disciplinaries]",
		"You do not have permission to update staff disciplinary types [manage_disciplinaries]",
		"You do not have permission to delete staff disciplinary types [manage_disciplinaries]",

		// Shop and vote credit tiers.
		"Cents cannot be lower than 0",
		"Votes cannot be lower than 0",
		"Duration cannot be lower than 0",
		"Target type must be either 'bot' or 'server'",
		"Max uses must be greater than 0",
		"Reuse wait duration must be greater than 0",
		"Expiry must be greater than 0",
		"Cannot delete benefit as it is used by shop items",
		"Benefit %s does not exist",
		"Item %q does not exist",
		"You do not have permission to create vote credit tiers [manage_shop]",
		"You do not have permission to update vote credit tiers [manage_shop]",
		"You do not have permission to delete vote credit tiers [manage_shop]",
		"You do not have permission to create shop items [manage_shop]",
		"You do not have permission to update shop items [manage_shop]",
		"You do not have permission to delete shop items [manage_shop]",
		"You do not have permission to create shop item benefits [manage_shop]",
		"You do not have permission to update shop item benefits [manage_shop]",
		"You do not have permission to delete shop item benefits [manage_shop]",
		"You do not have permission to list shop coupons [view_shop]",
		"You do not have permission to create shop coupons [manage_shop]",
		"You do not have permission to update shop coupons [manage_shop]",
		"You do not have permission to delete shop coupons [manage_shop]",

		// Bot whitelist uses PARENTHESES, unlike every other message.
		"You do not have permission to add to the bot whitelist (bot_whitelist.create)",
		"You do not have permission to update bot whitelist (bot_whitelist.update)",
		"You do not have permission to delete bot whitelist entries (bot_whitelist.delete)",

		// Misc.
		"Searching this target type is not implemented",
		"You do not have permission to view rpc logs [view_audit_logs]",
		"Invalid method",
		"Path must start with /",

		// Hello instance config.
		"Arcadia Staging Panel Instance",
		"Arcadia Production Panel Instance",
	}

	assertContains(t, "../panel", golden)
}

// TestGoldenRPCStrings pins the RPC layer's messages and mod-log embed text,
// including the titles whose leading spaces are intentional by accident.
func TestGoldenRPCStrings(t *testing.T) {
	golden := []string{
		// Pipeline.
		"This method does not support this target type yet",
		"You need the %s permission to use %s",
		"Rate limit exceeded. Wait 5-10 minutes and try again?",
		"Failed to get ratelimit count",
		"Failed to reset user token",
		"Reason must be lower than/equal to %d characters",

		// Handler errors. The leading spaces are upstream's.
		" does not exist",
		"This bot is not pending review",
		"This bot is a test bot",
		"This bot is already claimed by <@%s>",
		"<@%s> is not claimed",
		"Entity is not pending review?",
		" is not pending review?",
		"<@%s> is not claimed? Do ``/claim`` to claim this bot first!",
		"Certified bots cannot be unverified",
		"You can't force delete this bot with 'kick' enabled!",
		"Invalid team ID",
		" is in a team. Please use BotTransferOwnershipTeam",
		" is not in a team. Please use TransferOwnership",

		// Embed titles, with their leading spaces.
		" Claimed!",
		" Unclaimed!",
		" Approved!",
		" Denied!",
		"__ Unverified For Futher Review!__",
		"Premium Added!",
		"Premium Removed!",
		"Vote Ban Edit!",
		"Vote Ban Removed!",
		"__Entity Vote Reset!__",
		"__All Entity Votes Reset!__",
		" Force Deleted!",
		" Force Certified!",
		" Uncertified!",
		" Ownership Force Update!",
		"[Apps] Banned User",
		"[Apps] Unbanned User",

		// Embed footers.
		"This is completely normal, don't worry!",
		"Well done, young traveller!",
		"Well done, young traveller at getting denied from the club!",
		"Gonna be pending further review...",
		"Well done, young traveller! Use it wisely...",
		"Well done, young traveller. Sad to see you go...",
		"Remember: don't abuse our services!",
		"Sad life :(",
		"Neat",
		"Uh oh, looks like you've been naughty...",
		"Contact support if you think this is a mistake",
		"Welcome, back!",
	}

	assertContains(t, "../rpc", golden)
}

// TestGoldenTaskStrings pins the background tasks' announcements.
func TestGoldenTaskStrings(t *testing.T) {
	golden := []string{
		"Auto-Unclaimed Bot",
		"Bot Unclaimed!",
		"This is completely normal, don't worry!",
		"User Ban",
		"User Unban",
		"Bot Deleted From Discord!",
		"If this is a mistake, please contact support!",
		"__Automated Per-Monthly Vote Reset!__",
		"Welcome back :)",
		"**Weekly Job**\nSynced Top Reviewers!",
		"Staff Permissions Resync",
		"Old Positions",
		"New Positions",
		"Old Permissions",
		"New Permissions",
		"Syncing top reviewers, weekly job.",
		"Removing corresponding role",
		"Adding corresponding role",
		"Failed to get staff guild for staff perms resync",
		"has been removed from the premium list because it is not/no longer approved or certified.",
		"has been removed from the premium list as their subscription has expired.",
	}

	assertContains(t, "../tasks", golden)
}

// TestGoldenBotStrings pins the Discord bot's replies.
func TestGoldenBotStrings(t *testing.T) {
	golden := []string{
		"You are not staff",
		"You are not in the testing server",
		"You are not in the main server",
		"You are not in the staff server",
		"There was an error running this command: %s",
		"Whoa there, do you have permission to do this?: %s",
		"This command is currently disabled",
		"Some available options are ``staff list``, ``staff guildlist``, ``staff_guildleave``, ``staff_stats``",
		"Claimed bot successfully, the bot owner has been informed",
		"Unclaimed bot successfully!",
		"Okay! The bot has been denied.",
		"Approved bot!\n%s",
		"RPC did not return as expected???",
		"There are no bots in the queue!",
		"*You are free to test this bot. It is not claimed*",
		"Bot Already Claimed",
		"Force Claim",
		"Remind Reviewer",
		"did you forgot to finish testing",
		"The staff guide can be found at %s/staff/guide. Please **do not** bookmark this page as the URL may change in the future",
		"Invite: %s",
		"Invite (DB, Unsafe)",
		"View Page",
		"Removed guild",
		"You need the %s permission to use this command",
		"**Force Refresh**\nSynced Top Reviewers!",
		"You are not the owner/additional owner of any bots",
		"You are not the owner/additional owner of any approved or certified bots",
		"You are the owner/additional owner of a certified bot! Giving you certified role",
		"You are the owner/additional owner of an approved bot! Giving you approved role",
		"Autorole due to bots owned",
		"You are not in the server",
		"Omniplex Analytics",
		"I hope it's good :eyes:",
		"Oh, hello there! Let's see who's been fighting bots the most :eyes:\n\n",
		"Staff Leaderboard",
		// RPC modal coercion.
		"Invalid boolean",
		"Invalid number",
		"Invalid time format. Unit must be years, months, weeks, days or hours",
		"Invalid time format. Format must be WITH A SPACE BETWEEN THE NUMBER AND THE UNIT",
		// explainme.
		"Welcome to IBL, ``explainme`` is a easy way to get an explanation on commands.",
	}

	assertContains(t, "../bot", golden)
}

// assertContains checks that every golden string appears as a literal somewhere
// in the package's non-test sources.
func assertContains(t *testing.T, dir string, golden []string) {
	t.Helper()

	source := readSources(t, dir)

	for _, want := range golden {
		if !strings.Contains(source, quoteFor(want)) {
			t.Errorf("frozen string missing from %s: %q", dir, want)
		}
	}
}

func readSources(t *testing.T, dir string) string {
	t.Helper()

	entries, err := os.ReadDir(dir)

	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	var sb strings.Builder

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))

		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		sb.Write(data)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// quoteFor renders a string the way it appears inside a Go literal, so golden
// strings containing quotes, newlines or backslashes still match.
func quoteFor(s string) string {
	return strings.Trim(strconv.Quote(s), `"`)
}
