package perms

import (
	"testing"

	"popplio/config"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
)

// withOwners points the config at a fixed owner list for the duration of a test.
func withOwners(t *testing.T, ids ...snowflake.ID) {
	t.Helper()

	previous := state.Config
	t.Cleanup(func() { state.Config = previous })

	state.Config = &config.Config{Arcadia: config.Arcadia{Owners: ids}}
}

func TestIsConfigOwner(t *testing.T) {
	// A nil config must not panic and must not make anyone an owner: the check
	// runs on every permission load, including before setup in tests.
	state.Config = nil

	if IsConfigOwner("510065483693817867") {
		t.Error("nobody is an owner without a config")
	}

	withOwners(t, snowflake.ID(510065483693817867), snowflake.ID(728871946456137770))

	if !IsConfigOwner("510065483693817867") || !IsConfigOwner("728871946456137770") {
		t.Error("a listed owner should be recognised")
	}

	if IsConfigOwner("123456789") {
		t.Error("an unlisted user should not be an owner")
	}

	if IsConfigOwner("") {
		t.Error("an empty user id should not be an owner")
	}
}

func TestOwnerHoldsEverything(t *testing.T) {
	owner := StaffGrants{ConfigOwner: true}

	if !owner.Resolve().Has(StaffForceRemoveBots) || !owner.Resolve().Has(StaffManageStaffRoles) {
		t.Error("an owner should hold every staff permission")
	}

	if !owner.Resolve().IsSuper() {
		t.Error("an owner should read as super")
	}

	// The bypass is additive: a member's real roles still resolve.
	withRoles := StaffGrants{
		Roles:       []Role{{ID: "reviewer", Index: 9, Perms: []Perm{StaffReviewBots}}},
		ConfigOwner: true,
	}

	set := withRoles.Resolve()

	if !set.Has(StaffReviewBots) || !set.Has(StaffManageShop) {
		t.Errorf("an owner with roles should hold both, got %v", set.Strings())
	}

	// It stays scoped to the staff domain: owning the instance is not owning
	// somebody's team.
	if Entity.NewSet().Has(EntityDeleteBots) {
		t.Error("the owner bypass must not leak into entity permissions")
	}
}

func TestOwnerOutranksEveryone(t *testing.T) {
	owner := StaffGrants{ConfigOwner: true}

	if owner.Rank() != OwnerRank {
		t.Errorf("Rank() = %d, want OwnerRank", owner.Rank())
	}

	// Every rank check in the codebase is "is the target's index at or above
	// mine", so the owner's rank has to be below the most senior role possible.
	mostSenior := StaffGrants{Roles: []Role{{ID: "owner_role", Index: 1}}}

	if owner.Rank() >= mostSenior.Rank() {
		t.Error("an owner must outrank the most senior role")
	}

	if owner.Rank() >= NoRank {
		t.Error("an owner must outrank a member with no roles")
	}

	// An owner who also holds roles is still ranked as an owner.
	both := StaffGrants{Roles: []Role{{ID: "intern", Index: 11}}, ConfigOwner: true}

	if both.Rank() != OwnerRank {
		t.Errorf("Rank() = %d for an owner holding a junior role, want OwnerRank", both.Rank())
	}
}

func TestOwnerCanManageAnything(t *testing.T) {
	owner := StaffGrants{ConfigOwner: true}.Resolve()

	// Granting and revoking anything, including what they were never given
	// explicitly, has to pass.
	err := CheckPatch(owner, Staff.NewSet(), Staff.NewSet(StaffAdministrator, StaffForceRemoveBots, StaffManagePremium))

	if err != nil {
		t.Errorf("an owner should be able to grant anything: %v", err)
	}

	err = CheckPatch(owner, Staff.NewSet(StaffAdministrator), Staff.NewSet())

	if err != nil {
		t.Errorf("an owner should be able to revoke anything: %v", err)
	}
}
