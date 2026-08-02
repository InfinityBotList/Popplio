package cdnpath

import (
	"testing"

	perms "github.com/infinitybotlist/kittycat/go"
)

func TestValidateName(t *testing.T) {
	ok := []string{"", "file.webp", "a b-c_d.e:f%g$h[i](j){k}@l!", "1234", "a.b.c"}
	bad := []string{".hidden", "a/b", `a\b`, "naïve", "a\tb", "../x", "a\nb"}

	for _, name := range ok {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}

	for _, name := range bad {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want an error", name)
		}
	}
}

func TestValidatePath(t *testing.T) {
	ok := []string{"", "a", "a/b", "a/b c", "a-b_c.d:e%f$g", "avatars/bots"}
	bad := []string{"../a", "a//b", `a\b`, "/a", "a/../b", "naïve", "a/..", "..", "a/b/../../.."}

	for _, p := range ok {
		if err := ValidatePath(p); err != nil {
			t.Errorf("ValidatePath(%q) = %v, want nil", p, err)
		}
	}

	for _, p := range bad {
		if err := ValidatePath(p); err == nil {
			t.Errorf("ValidatePath(%q) = nil, want an error", p)
		}
	}
}

// The error text is user-visible and frozen.
func TestValidatorMessages(t *testing.T) {
	if got := ErrBadName.Error(); got != "Asset name cannot contain disallowed characters, slashes or backslashes or start with a dot" {
		t.Errorf("ErrBadName = %q", got)
	}

	if got := ErrBadPath.Error(); got != "Asset path cannot contain non-ASCII characters, dot-dots, doubleslashes, backslashes or start with a slash" {
		t.Errorf("ErrBadPath = %q", got)
	}
}

// ContainedInScope must accept everything legitimate and reject every escape.
func TestContainedInScope(t *testing.T) {
	const root = "/silverpelt/cdn/ibl"

	inside := []string{
		root,
		root + "/avatars",
		root + "/avatars/bots/1.webp",
		root + "/a/./b",
		root + "/a/b/../c",
	}

	outside := []string{
		"/silverpelt/cdn/other",
		root + "/../other",
		"/etc/passwd",
		root + "x",
		"/",
	}

	for _, p := range inside {
		if !ContainedInScope(root, p) {
			t.Errorf("ContainedInScope(%q, %q) = false, want true", root, p)
		}
	}

	for _, p := range outside {
		if ContainedInScope(root, p) {
			t.Errorf("ContainedInScope(%q, %q) = true, want false", root, p)
		}
	}
}

func TestHasPerm(t *testing.T) {
	scoped := perms.PFSS([]string{"cdn#ibl@main.add_file"})
	blanket := perms.PFSS([]string{"cdn.add_file"})
	unrelated := perms.PFSS([]string{"cdn.delete"})
	none := perms.PFSS(nil)

	cases := []struct {
		name  string
		perms []perms.Permission
		scope string
		op    string
		want  bool
	}{
		{"scope-specific grant", scoped, "ibl@main", "add_file", true},
		{"blanket grant", blanket, "ibl@main", "add_file", true},
		{"blanket grant, any scope", blanket, "ibl@other", "add_file", true},
		{"wrong scope", scoped, "ibl@other", "add_file", false},
		{"wrong op", unrelated, "ibl@main", "add_file", false},
		{"no perms", none, "ibl@main", "add_file", false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPerm(tt.perms, tt.scope, tt.op); got != tt.want {
				t.Errorf("HasPerm() = %v, want %v", got, tt.want)
			}
		})
	}
}
