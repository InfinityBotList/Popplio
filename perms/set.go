package perms

import (
	"encoding/json"
	"maps"
	"slices"
)

// Set is a holder's effective permissions.
//
// The zero Set grants nothing and is safe to use. Sets share their backing map,
// so [Set.Clone] before mutating one a caller still holds; the [Set.With] and
// [Set.Without] helpers clone for you.
type Set struct {
	cat  *Catalogue
	held map[Perm]struct{}
}

// NewSet builds a set in this catalogue's domain.
//
// Permissions the catalogue does not declare are kept rather than dropped: a
// permission belonging to another service, or one added by a newer deployment,
// must survive being read and written back by this one.
func (c *Catalogue) NewSet(list ...Perm) Set {
	s := Set{cat: c, held: make(map[Perm]struct{}, len(list))}

	for _, p := range list {
		s.held[p] = struct{}{}
	}

	return s
}

// SetFromStrings builds a set from stored names.
func (c *Catalogue) SetFromStrings(list []string) Set {
	return c.NewSet(ParseStrings(list)...)
}

// Role is one source of permissions: a staff position, which is bound to a
// Discord role in the staff server.
//
// Index ranks roles against each other, lower being more senior. It has no
// effect on which permissions resolve — the union does not care about order —
// and exists so that management operations can refuse to let someone edit a
// role or a member at or above their own rank.
type Role struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Index int32  `json:"index"`
	Perms []Perm `json:"perms"`
}

// Resolve is every permission a holder has: the union of their roles'
// permissions and whatever extras are granted to them directly.
func (c *Catalogue) Resolve(roles []Role, extras ...Perm) Set {
	size := len(extras)

	for _, r := range roles {
		size += len(r.Perms)
	}

	s := Set{cat: c, held: make(map[Perm]struct{}, size)}

	for _, r := range roles {
		for _, p := range r.Perms {
			s.held[p] = struct{}{}
		}
	}

	for _, p := range extras {
		s.held[p] = struct{}{}
	}

	return s
}

// ResolveStrings is [Catalogue.Resolve] for a single flat source, which is what
// team members and API sessions have.
func (c *Catalogue) ResolveStrings(list []string) Set {
	return c.SetFromStrings(list)
}

// Has reports whether the set allows a permission. The domain's super
// permission allows everything.
func (s Set) Has(p Perm) bool {
	if len(s.held) == 0 {
		return false
	}

	if s.cat != nil {
		if _, ok := s.held[s.cat.Super]; ok {
			return true
		}
	}

	_, ok := s.held[p]

	return ok
}

// HasAll reports whether every permission is allowed.
func (s Set) HasAll(list ...Perm) bool {
	for _, p := range list {
		if !s.Has(p) {
			return false
		}
	}

	return true
}

// HasAny reports whether at least one of the permissions is allowed.
func (s Set) HasAny(list ...Perm) bool {
	for _, p := range list {
		if s.Has(p) {
			return true
		}
	}

	return false
}

// IsSuper reports whether the set holds the domain's super permission.
func (s Set) IsSuper() bool {
	if s.cat == nil {
		return false
	}

	_, ok := s.held[s.cat.Super]

	return ok
}

// contains reports whether a permission is present in its own right, ignoring
// the super permission. Management operations need this exact reading; access
// checks never do.
func (s Set) contains(p Perm) bool {
	_, ok := s.held[p]
	return ok
}

// Len is the number of permissions held.
func (s Set) Len() int {
	return len(s.held)
}

// IsEmpty reports whether the set holds nothing.
func (s Set) IsEmpty() bool {
	return len(s.held) == 0
}

// All returns the permissions held, catalogue order first and anything
// undeclared after it, so that output is stable and reads in the order the
// catalogue presents.
func (s Set) All() []Perm {
	out := make([]Perm, 0, len(s.held))

	if s.cat != nil {
		for _, d := range s.cat.defs {
			if _, ok := s.held[d.ID]; ok {
				out = append(out, d.ID)
			}
		}
	}

	extra := make([]Perm, 0)

	for p := range s.held {
		if s.cat != nil {
			if _, declared := s.cat.index[p]; declared {
				continue
			}
		}

		extra = append(extra, p)
	}

	slices.Sort(extra)

	return append(out, extra...)
}

// Undeclared returns the permissions held that this catalogue does not declare,
// which is how an interface can show that a holder carries something owned by
// another service.
func (s Set) Undeclared() []Perm {
	if s.cat == nil {
		return nil
	}

	var out []Perm

	for p := range s.held {
		if _, ok := s.cat.index[p]; !ok {
			out = append(out, p)
		}
	}

	slices.Sort(out)

	return out
}

// Strings renders the set for storage or a JSON response. It is never nil.
func (s Set) Strings() []string {
	return Strings(s.All())
}

// With returns a copy with more permissions granted.
func (s Set) With(list ...Perm) Set {
	out := s.Clone()

	if out.held == nil {
		out.held = make(map[Perm]struct{}, len(list))
	}

	for _, p := range list {
		out.held[p] = struct{}{}
	}

	return out
}

// Without returns a copy with permissions removed.
func (s Set) Without(list ...Perm) Set {
	out := s.Clone()

	for _, p := range list {
		delete(out.held, p)
	}

	return out
}

// Clone returns a copy that can be mutated without affecting the original.
func (s Set) Clone() Set {
	return Set{cat: s.cat, held: maps.Clone(s.held)}
}

// Equal reports whether two sets hold exactly the same permissions.
func (s Set) Equal(other Set) bool {
	return maps.Equal(s.held, other.held)
}

// Diff returns the permissions that differ between two sets: what would have to
// be granted or revoked to turn s into other. It is what [CheckPatch] audits.
func (s Set) Diff(other Set) []Perm {
	changed := make([]Perm, 0)

	for p := range s.held {
		if _, ok := other.held[p]; !ok {
			changed = append(changed, p)
		}
	}

	for p := range other.held {
		if _, ok := s.held[p]; !ok {
			changed = append(changed, p)
		}
	}

	slices.Sort(changed)

	return changed
}

// MarshalJSON encodes the set as the array of permission names the API and the
// staff panel expect.
func (s Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Strings())
}
