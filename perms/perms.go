// Package perms implements Popplio's permission model.
//
// A permission is a flat, self-describing name — `review_bots`, `manage_shop`,
// `edit_team_members`. There are no namespaces, no wildcards, no negations and
// no precedence rules: a holder either has a permission or does not.
//
// # Where permissions come from
//
// Permissions are always granted by a role, plus optional per-holder extras:
//
//   - Staff roles live in `staff_positions` and are bound to a Discord role in
//     the staff server. The resync task mirrors role membership into
//     `staff_members.positions`, so granting someone a Discord role grants them
//     that role's permissions. Extras are `staff_members.perm_overrides`.
//   - Team permissions live in `team_members.flags`, and API session scopes in
//     `api_sessions.perm_limits`. Neither has roles yet, so they are all extras.
//
// A holder's effective permissions are the union of every source. Nothing
// subtracts. That is the whole model, and it is why a permission check is a
// single map lookup.
//
// The one exception is who may hold staff permissions at all: a Discord bot
// account holds none, from any source, including the owners list. See
// [ErrBotAccount].
//
// # Catalogues
//
// Every permission is declared in a [Catalogue]: [Staff] for what staff can do
// on the platform, [Entity] for what a team member can do with a team's bots,
// servers and settings. The catalogue is what makes the model manageable — it
// carries the human name, description and category shown in the staff bot, the
// panel and the public API, and it is what write paths validate against.
//
// # Super permissions
//
// Each catalogue names one permission that implies every other:
// [StaffAdministrator] and [EntityOwner]. It is the only implication in the
// system.
//
// # Migrating from the previous model
//
// This replaces a `namespace.action` scheme with negators (`~rpc.Approve`),
// wildcards (`rpc.*`), an `@clear` marker and role-index precedence. Each old
// permission's replacement is recorded in [Definition.Legacy]; `exp/rewrite/flatperms.sql`
// applies exactly that mapping to the database.
package perms

import (
	"fmt"
	"slices"
	"strings"
)

// Perm is a permission name: lowercase, snake_case, no punctuation.
type Perm string

// String renders the permission for logs, errors and storage.
func (p Perm) String() string {
	return string(p)
}

// Definition describes one permission. It is the single place a permission's
// name, meaning and grouping are written down, and it feeds the staff bot, the
// staff panel and the public API alike.
type Definition struct {
	// ID is the stored name.
	ID Perm `json:"id"`
	// Name is the human label, e.g. "Review Bots".
	Name string `json:"name"`
	// Description says what holding it lets someone do.
	Description string `json:"description"`
	// Category groups related permissions for display.
	Category string `json:"category"`
	// Dangerous marks permissions that hand over broad or irreversible power,
	// so that interfaces can warn before granting them.
	Dangerous bool `json:"dangerous,omitempty"`
	// Legacy lists the `namespace.action` permissions this one replaces. It
	// documents the migration and is not consulted at runtime.
	Legacy []string `json:"-"`
}

// Catalogue is the set of permissions that exist in one domain.
type Catalogue struct {
	// Domain names the catalogue in errors, e.g. "staff".
	Domain string
	// Super is the permission that implies every other one in the domain.
	Super Perm

	defs  []Definition
	index map[Perm]Definition
}

// NewCatalogue declares a domain's permissions. It panics on a duplicate or on
// a super permission that is not itself declared, both of which are programming
// errors that would otherwise surface as silently missing permissions.
func NewCatalogue(domain string, super Perm, defs []Definition) *Catalogue {
	c := &Catalogue{
		Domain: domain,
		Super:  super,
		defs:   defs,
		index:  make(map[Perm]Definition, len(defs)),
	}

	for _, d := range defs {
		if _, dup := c.index[d.ID]; dup {
			panic(fmt.Sprintf("perms: duplicate permission %q in the %s catalogue", d.ID, domain))
		}

		c.index[d.ID] = d
	}

	if _, ok := c.index[super]; !ok {
		panic(fmt.Sprintf("perms: super permission %q is not declared in the %s catalogue", super, domain))
	}

	return c
}

// Definitions returns every permission in the catalogue, in declaration order,
// which is the order interfaces should present them in.
func (c *Catalogue) Definitions() []Definition {
	return slices.Clone(c.defs)
}

// Categories returns the distinct categories in declaration order.
func (c *Catalogue) Categories() []string {
	var out []string

	for _, d := range c.defs {
		if !slices.Contains(out, d.Category) {
			out = append(out, d.Category)
		}
	}

	return out
}

// InCategory returns the permissions of one category, in declaration order.
func (c *Catalogue) InCategory(category string) []Definition {
	var out []Definition

	for _, d := range c.defs {
		if strings.EqualFold(d.Category, category) {
			out = append(out, d)
		}
	}

	return out
}

// Lookup returns a permission's definition.
func (c *Catalogue) Lookup(p Perm) (Definition, bool) {
	d, ok := c.index[p]
	return d, ok
}

// Label is the human name of a permission, falling back to the raw name for
// one this catalogue does not declare.
func (c *Catalogue) Label(p Perm) string {
	if d, ok := c.index[p]; ok {
		return d.Name
	}

	return string(p)
}

// Validate rejects a permission this catalogue does not declare. Write paths
// use it so a typo is refused at the edge instead of being stored as a
// permission that can never match.
func (c *Catalogue) Validate(p Perm) error {
	if _, ok := c.index[p]; !ok {
		return fmt.Errorf("unknown %s permission %q", c.Domain, p)
	}

	return nil
}

// ValidateStrings validates a list on its way into the database, reporting
// every unknown name at once rather than one per attempt.
func (c *Catalogue) ValidateStrings(list []string) error {
	var unknown []string

	for _, s := range list {
		if _, ok := c.index[Perm(s)]; !ok {
			unknown = append(unknown, s)
		}
	}

	switch len(unknown) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("unknown %s permission %q", c.Domain, unknown[0])
	default:
		return fmt.Errorf("unknown %s permissions: %s", c.Domain, strings.Join(unknown, ", "))
	}
}

// Suggest returns the declared permissions whose name or label contains the
// given text, so an interface can answer "did you mean" without the caller
// knowing the catalogue.
func (c *Catalogue) Suggest(text string) []Definition {
	text = strings.ToLower(strings.TrimSpace(text))

	if text == "" {
		return nil
	}

	var out []Definition

	for _, d := range c.defs {
		if strings.Contains(string(d.ID), text) || strings.Contains(strings.ToLower(d.Name), text) {
			out = append(out, d)
		}
	}

	return out
}

// ParseStrings converts stored names into permissions.
func ParseStrings(list []string) []Perm {
	out := make([]Perm, 0, len(list))

	for _, s := range list {
		out = append(out, Perm(s))
	}

	return out
}

// Strings renders permissions for storage or a JSON response. It is never nil,
// as the API contract uses `[]` rather than `null` for an empty list.
func Strings(list []Perm) []string {
	out := make([]string, 0, len(list))

	for _, p := range list {
		out = append(out, string(p))
	}

	return out
}
