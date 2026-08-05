package perms

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

// A bot holds nothing, from any source, in any combination. This is the whole
// rule, so it is tested source by source rather than once.
func TestBotAccountHoldsNothing(t *testing.T) {
	roles := []Role{{ID: "admins", Index: 1, Perms: []Perm{StaffAdministrator}}}

	cases := []struct {
		name   string
		grants StaffGrants
	}{
		{"roles", StaffGrants{Roles: roles, BotAccount: true}},
		{"direct grants", StaffGrants{Extras: []Perm{StaffReviewBots, StaffViewPanel}, BotAccount: true}},
		{"the owners list", StaffGrants{ConfigOwner: true, BotAccount: true}},
		{"everything at once", StaffGrants{
			Roles:       roles,
			Extras:      []Perm{StaffReviewBots},
			ConfigOwner: true,
			BotAccount:  true,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := tc.grants.Resolve()

			if !resolved.IsEmpty() {
				t.Errorf("a bot resolved to %v", resolved.Strings())
			}

			if resolved.IsSuper() {
				t.Error("a bot must never read as super")
			}

			for _, p := range []Perm{StaffAdministrator, StaffReviewBots, StaffViewPanel, StaffViewStaff} {
				if resolved.Has(p) {
					t.Errorf("a bot should not hold %s", p)
				}
			}
		})
	}
}

// Rank is what the "can I edit them" checks compare against, so a bot must not
// inherit the standing its roles or the owners list would otherwise give it.
func TestBotAccountHasNoRank(t *testing.T) {
	withOwners(t, snowflake.ID(510065483693817867))

	bot := StaffGrants{
		Roles:       []Role{{ID: "admins", Index: 1, Perms: []Perm{StaffAdministrator}}},
		ConfigOwner: true,
		BotAccount:  true,
	}

	if got := bot.Rank(); got != NoRank {
		t.Errorf("Rank() = %d, want NoRank (%d)", got, NoRank)
	}

	// The same grants on a person keep the standing the bot was denied, so the
	// test is about the flag and not about the grants being empty.
	person := bot
	person.BotAccount = false

	if got := person.Rank(); got != OwnerRank {
		t.Errorf("a person with these grants should rank %d, got %d", OwnerRank, got)
	}
}

// A permission cannot be handed to a bot by way of a patch check either: with
// nothing resolved, there is nothing it is allowed to keep.
func TestBotAccountCannotBePatchedInto(t *testing.T) {
	bot := StaffGrants{BotAccount: true, Extras: []Perm{StaffReviewBots}}

	current := bot.Resolve()
	next := current.With(StaffReviewBots)

	// The manager holds the permission, so the patch itself is legitimate; what
	// makes it meaningless is that the bot's resolved set stays empty.
	manager := Staff.NewSet(StaffAdministrator)

	if err := CheckPatch(manager, current, next); err != nil {
		t.Fatalf("the patch should be legitimate for the manager: %v", err)
	}

	if !bot.Resolve().IsEmpty() {
		t.Error("writing a grant onto a bot must not resolve into anything")
	}
}
