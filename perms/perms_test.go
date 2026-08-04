package perms

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestCatalogueIsWellFormed(t *testing.T) {
	for _, c := range []*Catalogue{Staff, Entity} {
		seenLegacy := map[string]Perm{}

		for _, d := range c.Definitions() {
			if d.Name == "" || d.Description == "" || d.Category == "" {
				t.Errorf("%s: %s is missing a name, description or category", c.Domain, d.ID)
			}

			if strings.ContainsAny(string(d.ID), ".*~ #@") {
				t.Errorf("%s: %s is not a flat name", c.Domain, d.ID)
			}

			if strings.ToLower(string(d.ID)) != string(d.ID) {
				t.Errorf("%s: %s is not lowercase", c.Domain, d.ID)
			}

			// A legacy permission must map to exactly one replacement, or the
			// migration would have to guess.
			for _, l := range d.Legacy {
				if prev, dup := seenLegacy[l]; dup {
					t.Errorf("%s: legacy permission %s maps to both %s and %s", c.Domain, l, prev, d.ID)
				}

				seenLegacy[l] = d.ID
			}
		}
	}
}

func TestHas(t *testing.T) {
	set := Staff.NewSet(StaffReviewBots, StaffViewApps)

	if !set.Has(StaffReviewBots) {
		t.Error("a held permission should be allowed")
	}

	if set.Has(StaffManageShop) {
		t.Error("an unheld permission should not be allowed")
	}

	if set.IsSuper() {
		t.Error("a plain set is not super")
	}

	admin := Staff.NewSet(StaffAdministrator)

	if !admin.Has(StaffManageShop) || !admin.Has(StaffForceRemoveBots) {
		t.Error("administrator should imply every staff permission")
	}

	if !admin.IsSuper() {
		t.Error("administrator is the staff super permission")
	}

	// The super permission is per domain: an entity owner is not a staff admin.
	owner := Entity.NewSet(EntityOwner)

	if !owner.Has(EntityDeleteBots) {
		t.Error("owner should imply every entity permission")
	}

	if (Set{}).Has(StaffReviewBots) {
		t.Error("the zero set grants nothing")
	}
}

func TestResolveIsAUnion(t *testing.T) {
	roles := []Role{
		{ID: "reviewer", Index: 9, Perms: []Perm{StaffReviewBots, StaffViewPanel}},
		{ID: "manager", Index: 3, Perms: []Perm{StaffManageShop, StaffViewPanel}},
	}

	set := Staff.Resolve(roles, StaffViewTickets)

	for _, p := range []Perm{StaffReviewBots, StaffViewPanel, StaffManageShop, StaffViewTickets} {
		if !set.Has(p) {
			t.Errorf("union should include %s", p)
		}
	}

	if set.Has(StaffManagePremium) {
		t.Error("union should not invent permissions")
	}

	if set.Len() != 4 {
		t.Errorf("duplicate permissions should collapse, got %v", set.Strings())
	}

	// Order of roles cannot change the outcome — that is the point of the model.
	slices.Reverse(roles)

	if !set.Equal(Staff.Resolve(roles, StaffViewTickets)) {
		t.Error("resolution must not depend on role order")
	}
}

func TestUndeclaredPermissionsSurvive(t *testing.T) {
	set := Staff.SetFromStrings([]string{"review_bots", "some_other_service_perm"})

	if !slices.Contains(set.Strings(), "some_other_service_perm") {
		t.Error("a permission this catalogue does not declare must survive a round trip")
	}

	if got := set.Undeclared(); len(got) != 1 || got[0] != "some_other_service_perm" {
		t.Errorf("Undeclared() = %v, want [some_other_service_perm]", got)
	}

	if !set.Has(StaffReviewBots) {
		t.Error("declared permissions still work alongside undeclared ones")
	}
}

func TestStrings(t *testing.T) {
	// Catalogue order first, undeclared names after, so output is stable.
	set := Staff.SetFromStrings([]string{"zzz_unknown", "manage_shop", "administrator", "review_bots"})

	want := []string{"administrator", "review_bots", "manage_shop", "zzz_unknown"}

	if got := set.Strings(); !slices.Equal(got, want) {
		t.Errorf("Strings() = %v, want %v", got, want)
	}

	if got := Staff.NewSet().Strings(); got == nil || len(got) != 0 {
		t.Errorf("an empty set must render as [], got %v", got)
	}
}

func TestValidate(t *testing.T) {
	if err := Staff.ValidateStrings([]string{"review_bots", "manage_shop"}); err != nil {
		t.Errorf("declared permissions should validate: %v", err)
	}

	err := Staff.ValidateStrings([]string{"review_bots", "rpc.Approve", "bogus"})

	if err == nil {
		t.Fatal("unknown permissions should be rejected")
	}

	// Every unknown name is reported at once.
	for _, want := range []string{"rpc.Approve", "bogus"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}

	// An old-style permission is not silently accepted anywhere.
	if err := Entity.Validate("bot.add"); err == nil {
		t.Error("entity catalogue should reject the old namespace.action form")
	}
}

func TestCheckPatch(t *testing.T) {
	manager := Entity.NewSet(EntityAddBots, EntityEditBots)

	if err := CheckPatch(manager, Entity.NewSet(), Entity.NewSet(EntityAddBots)); err != nil {
		t.Errorf("granting a held permission should be allowed: %v", err)
	}

	if err := CheckPatch(manager, Entity.NewSet(EntityAddBots), Entity.NewSet()); err != nil {
		t.Errorf("revoking a held permission should be allowed: %v", err)
	}

	err := CheckPatch(manager, Entity.NewSet(), Entity.NewSet(EntityDeleteBots))

	if !errors.Is(err, ErrCannotManage) {
		t.Errorf("granting an unheld permission should fail with ErrCannotManage, got %v", err)
	}

	// Revoking something the manager does not hold is equally off limits.
	err = CheckPatch(manager, Entity.NewSet(EntityDeleteBots), Entity.NewSet())

	if !errors.Is(err, ErrCannotManage) {
		t.Errorf("revoking an unheld permission should fail with ErrCannotManage, got %v", err)
	}

	if err := CheckPatch(Entity.NewSet(EntityOwner), Entity.NewSet(), Entity.NewSet(EntityDeleteBots, EntityManageSessions)); err != nil {
		t.Errorf("an owner should be able to grant anything: %v", err)
	}

	got := Unmanageable(manager, Entity.NewSet(), Entity.NewSet(EntityDeleteBots, EntityManageSessions, EntityAddBots))

	if len(got) != 2 {
		t.Errorf("Unmanageable() = %v, want the two unheld permissions", got)
	}
}

func TestRank(t *testing.T) {
	g := StaffGrants{Roles: []Role{
		{ID: "a", Index: 9},
		{ID: "b", Index: 2},
	}}

	if g.Rank() != 2 {
		t.Errorf("Rank() = %d, want the most senior role's index", g.Rank())
	}

	if (StaffGrants{}).Rank() != NoRank {
		t.Error("a member with no roles must rank below everyone")
	}
}

func TestSetMutationDoesNotAlias(t *testing.T) {
	base := Staff.NewSet(StaffReviewBots)
	grown := base.With(StaffManageShop)
	shrunk := grown.Without(StaffReviewBots)

	if base.Has(StaffManageShop) {
		t.Error("With must not modify the original")
	}

	if !grown.Has(StaffReviewBots) {
		t.Error("Without must not modify the original")
	}

	if shrunk.Has(StaffReviewBots) || !shrunk.Has(StaffManageShop) {
		t.Errorf("Without produced %v", shrunk.Strings())
	}
}

func BenchmarkHas(b *testing.B) {
	set := Staff.NewSet(
		StaffViewPanel, StaffReviewBots, StaffCertifyBots, StaffViewApps,
		StaffManageShop, StaffViewShop, StaffManagePartners, StaffViewStaff,
	)

	b.ResetTimer()

	for range b.N {
		if !set.Has(StaffManageShop) {
			b.Fatal("expected permission")
		}
	}
}

func BenchmarkResolve(b *testing.B) {
	roles := []Role{
		{ID: "a", Index: 9, Perms: []Perm{StaffReviewBots, StaffViewPanel, StaffViewApps}},
		{ID: "b", Index: 3, Perms: []Perm{StaffManageShop, StaffViewShop}},
		{ID: "c", Index: 1, Perms: []Perm{StaffManageStaffRoles, StaffManagePartners}},
	}

	b.ResetTimer()

	for range b.N {
		_ = Staff.Resolve(roles, StaffViewTickets)
	}
}
