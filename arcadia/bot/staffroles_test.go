package bot

import (
	"strings"
	"testing"

	"popplio/perms"

	"github.com/disgoorg/disgo/discord"
)

func TestResolvePermission(t *testing.T) {
	got, err := resolvePermission("review_bots")

	if err != nil || got != perms.StaffReviewBots {
		t.Errorf("resolvePermission(exact) = %q, %v", got, err)
	}

	// Backticks and case are what people actually type after copying a name out
	// of an embed.
	got, err = resolvePermission(" ``Review_Bots`` ")

	if err != nil || got != perms.StaffReviewBots {
		t.Errorf("resolvePermission(decorated) = %q, %v", got, err)
	}

	// A unique partial match is taken as what they meant.
	got, err = resolvePermission("whitelist")

	if err != nil || got != perms.StaffManageBotWhitelist {
		t.Errorf("resolvePermission(unique partial) = %q, %v", got, err)
	}

	// An ambiguous one lists the candidates instead of guessing.
	_, err = resolvePermission("manage_staff")

	if err == nil {
		t.Fatal("an ambiguous permission should not resolve")
	}

	for _, want := range []string{"manage_staff_members", "manage_staff_roles"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should suggest %q", err, want)
		}
	}

	// An old-style name is no longer a permission and must not silently work.
	if _, err := resolvePermission("rpc.Approve"); err == nil {
		t.Error("an old namespace.action permission should not resolve")
	}

	if _, err := resolvePermission(""); err == nil {
		t.Error("an empty permission should not resolve")
	}
}

func TestCanEditRole(t *testing.T) {
	manager := managerContext{rank: 3}

	if err := manager.canEditRole(staffRole{Name: "intern", Index: 9}); err != nil {
		t.Errorf("a more junior role should be editable: %v", err)
	}

	if err := manager.canEditRole(staffRole{Name: "owner", Index: 1}); err == nil {
		t.Error("a more senior role must not be editable")
	}

	// Equal rank is refused too: peers cannot rewrite each other's role.
	if err := manager.canEditRole(staffRole{Name: "peer", Index: 3}); err == nil {
		t.Error("a role at the caller's own rank must not be editable")
	}

	// Someone with no roles at all can edit nothing.
	none := managerContext{rank: perms.NoRank}

	if err := none.canEditRole(staffRole{Name: "intern", Index: 9}); err == nil {
		t.Error("a member with no roles must not be able to edit any role")
	}
}

func TestPermissionFields(t *testing.T) {
	fields := permissionFields(perms.Staff.NewSet())

	if len(fields) != 1 || fields[0].Value != "None" {
		t.Errorf("an empty set should render as None, got %+v", fields)
	}

	fields = permissionFields(perms.Staff.NewSet(perms.StaffAdministrator))

	if len(fields) != 1 || !strings.Contains(fields[0].Value, "everything") {
		t.Errorf("administrator should render as everything, got %+v", fields)
	}

	fields = permissionFields(perms.Staff.NewSet(perms.StaffReviewBots, perms.StaffManageShop))

	if len(fields) != 2 {
		t.Fatalf("two permissions in two categories should be two fields, got %d", len(fields))
	}

	// Permissions belonging to another service are shown, not hidden.
	fields = permissionFields(perms.Staff.SetFromStrings([]string{"review_bots", "some_other_service"}))

	var found bool

	for _, f := range fields {
		if strings.Contains(f.Value, "some_other_service") {
			found = true
		}
	}

	if !found {
		t.Error("an undeclared permission should still be shown")
	}
}

func TestChunkMessage(t *testing.T) {
	var sb strings.Builder

	for range 200 {
		sb.WriteString("a line of permission text that is reasonably long\n")
	}

	chunks := chunkMessage(sb.String(), 1900)

	if len(chunks) < 2 {
		t.Fatalf("long text should be split, got %d chunk(s)", len(chunks))
	}

	for _, chunk := range chunks {
		if len(chunk) > 1900 {
			t.Errorf("chunk of %d characters exceeds the limit", len(chunk))
		}
	}

	// Putting the chunks back together with the newline each split consumed must
	// reproduce the input exactly: no line lost, none duplicated.
	if rejoined := strings.Join(chunks, "\n"); rejoined != sb.String() {
		t.Errorf("rejoined text differs from the input (%d vs %d characters)", len(rejoined), sb.Len())
	}

	if got := chunkMessage("short", 1900); len(got) != 1 {
		t.Errorf("short text should stay in one chunk, got %d", len(got))
	}
}

// The commands are registered under names the help renderer and Discord both
// use, so a clash with an existing command would silently shadow one of them.
func TestStaffCommandsRegisterCleanly(t *testing.T) {
	registry = map[string]*Command{}
	ordered = nil

	registerCommands()

	for _, name := range []string{"staffroles", "staffperms", "permissions"} {
		cmd, ok := registry[name]

		if !ok {
			t.Fatalf("%s is not registered", name)
		}

		for _, sub := range cmd.Subcommands {
			if sub.Run == nil {
				t.Errorf("%s %s has no handler", name, sub.Name)
			}

			if sub.Description == "" {
				t.Errorf("%s %s has no description", name, sub.Name)
			}
		}
	}

	// Discord rejects a command whose description is over 100 characters, and
	// truncate() only protects the top level.
	for _, cmd := range ordered {
		for _, sub := range cmd.Subcommands {
			if len(sub.Description) > 100 {
				t.Errorf("%s %s has a %d character description", cmd.Name, sub.Name, len(sub.Description))
			}
		}
	}
}

// Discord rejects a whole command whose options put a required one after an
// optional one, and a rejected command is simply missing from the picker with
// nothing to show for it but a line in the startup log. The registration payload
// is checked here so that a bad option order fails the build instead.
func TestCommandOptionsRegisterInAValidOrder(t *testing.T) {
	registry = map[string]*Command{}
	ordered = nil

	registerCommands()

	var check func(path string, options []discord.ApplicationCommandOption)

	check = func(path string, options []discord.ApplicationCommandOption) {
		optionalSeen := ""

		for _, opt := range options {
			if sub, ok := opt.(discord.ApplicationCommandOptionSubCommand); ok {
				check(path+" "+sub.Name, sub.Options)
				continue
			}

			required := false

			switch o := opt.(type) {
			case discord.ApplicationCommandOptionString:
				required = o.Required
			case discord.ApplicationCommandOptionUser:
				required = o.Required
			case discord.ApplicationCommandOptionRole:
				required = o.Required
			case discord.ApplicationCommandOptionInt:
				required = o.Required
			case discord.ApplicationCommandOptionBool:
				required = o.Required
			}

			if required && optionalSeen != "" {
				t.Errorf("%s: required option %q comes after optional %q, which Discord rejects",
					path, opt.OptionName(), optionalSeen)
			}

			if !required {
				optionalSeen = opt.OptionName()
			}
		}
	}

	for _, cmd := range buildCommands() {
		slash, ok := cmd.(discord.SlashCommandCreate)

		if !ok {
			continue
		}

		check("/"+slash.Name, slash.Options)
	}
}

// The interactive editor is what these commands are mostly for, and Discord
// lists subcommands in the order they are registered.
func TestEditIsTheFirstSubcommand(t *testing.T) {
	registry = map[string]*Command{}
	ordered = nil

	registerCommands()

	for _, name := range []string{"staffroles", "staffperms"} {
		cmd, ok := registry[name]

		if !ok {
			t.Fatalf("%s is not registered", name)
		}

		if len(cmd.Subcommands) == 0 || cmd.Subcommands[0].Name != "edit" {
			t.Errorf("/%s should offer edit first, got %q", name, cmd.Subcommands[0].Name)
		}
	}
}
