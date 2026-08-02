package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// Every action union must round-trip both its unit variants (bare strings) and
// its struct variants (single-key objects).
func TestActionUnionsRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		wire string
		into func() json.Unmarshaler
	}{
		{"CdnAssetAction unit", `"ListPath"`, func() json.Unmarshaler { return &CdnAssetAction{} }},
		{"CdnAssetAction unit ReadFile", `"ReadFile"`, func() json.Unmarshaler { return &CdnAssetAction{} }},
		{"CdnAssetAction unit CreateFolder", `"CreateFolder"`, func() json.Unmarshaler { return &CdnAssetAction{} }},
		{"CdnAssetAction unit Delete", `"Delete"`, func() json.Unmarshaler { return &CdnAssetAction{} }},
		{"CdnAssetAction AddFile", `{"AddFile":{"overwrite":true,"chunks":["a"],"sha512":"x"}}`, func() json.Unmarshaler { return &CdnAssetAction{} }},
		{"CdnAssetAction CopyFile", `{"CopyFile":{"overwrite":false,"delete_original":true,"copy_to":"a/b"}}`, func() json.Unmarshaler { return &CdnAssetAction{} }},
		{"CdnAssetAction PersistGit", `{"PersistGit":{"message":"m","current_dir":false}}`, func() json.Unmarshaler { return &CdnAssetAction{} }},

		{"PartnerAction unit", `"List"`, func() json.Unmarshaler { return &PartnerAction{} }},
		{"PartnerAction Delete", `{"Delete":{"id":"x"}}`, func() json.Unmarshaler { return &PartnerAction{} }},

		{"BlogAction unit", `"ListEntries"`, func() json.Unmarshaler { return &BlogAction{} }},
		{"BlogAction DeleteEntry", `{"DeleteEntry":{"itag":"x"}}`, func() json.Unmarshaler { return &BlogAction{} }},

		{"ChangelogAction unit", `"ListEntries"`, func() json.Unmarshaler { return &ChangelogAction{} }},
		{"ChangelogAction DeleteEntry", `{"DeleteEntry":{"version":"1.0"}}`, func() json.Unmarshaler { return &ChangelogAction{} }},

		{"StaffPositionAction unit", `"ListPositions"`, func() json.Unmarshaler { return &StaffPositionAction{} }},
		{"StaffPositionAction SwapIndex", `{"SwapIndex":{"a":"1","b":"2"}}`, func() json.Unmarshaler { return &StaffPositionAction{} }},
		{"StaffPositionAction SetIndex", `{"SetIndex":{"id":"1","index":4}}`, func() json.Unmarshaler { return &StaffPositionAction{} }},
		{"StaffPositionAction DeletePosition", `{"DeletePosition":{"id":"1"}}`, func() json.Unmarshaler { return &StaffPositionAction{} }},

		{"StaffMemberAction unit", `"ListMembers"`, func() json.Unmarshaler { return &StaffMemberAction{} }},
		{"StaffMemberAction EditMember", `{"EditMember":{"user_id":"1","perm_overrides":["a.b"],"no_autosync":false,"unaccounted":true}}`, func() json.Unmarshaler { return &StaffMemberAction{} }},

		{"StaffDisciplinaryTypeAction unit", `"ListDisciplinaryTypes"`, func() json.Unmarshaler { return &StaffDisciplinaryTypeAction{} }},
		{"StaffDisciplinaryTypeAction Delete", `{"DeleteDisciplinaryType":{"id":"x"}}`, func() json.Unmarshaler { return &StaffDisciplinaryTypeAction{} }},

		{"VoteCreditTierAction unit", `"ListTiers"`, func() json.Unmarshaler { return &VoteCreditTierAction{} }},
		{"VoteCreditTierAction DeleteTier", `{"DeleteTier":{"id":"x"}}`, func() json.Unmarshaler { return &VoteCreditTierAction{} }},

		{"ShopItemAction unit", `"List"`, func() json.Unmarshaler { return &ShopItemAction{} }},
		{"ShopItemAction Delete", `{"Delete":{"id":"x"}}`, func() json.Unmarshaler { return &ShopItemAction{} }},

		{"ShopItemBenefitAction unit", `"List"`, func() json.Unmarshaler { return &ShopItemBenefitAction{} }},
		{"ShopItemBenefitAction Delete", `{"Delete":{"id":"x"}}`, func() json.Unmarshaler { return &ShopItemBenefitAction{} }},

		{"ShopCouponAction unit", `"List"`, func() json.Unmarshaler { return &ShopCouponAction{} }},
		{"ShopCouponAction Delete", `{"Delete":{"id":"x"}}`, func() json.Unmarshaler { return &ShopCouponAction{} }},

		{"BotWhitelistAction unit", `"List"`, func() json.Unmarshaler { return &BotWhitelistAction{} }},
		{"BotWhitelistAction Add", `{"Add":{"bot_id":"1","reason":"r"}}`, func() json.Unmarshaler { return &BotWhitelistAction{} }},
		{"BotWhitelistAction Delete", `{"Delete":{"bot_id":"1"}}`, func() json.Unmarshaler { return &BotWhitelistAction{} }},

		{"AuthorizeAction Begin", `{"Begin":{"scope":"s","redirect_url":"r"}}`, func() json.Unmarshaler { return &AuthorizeAction{} }},
		{"AuthorizeAction Logout", `{"Logout":{"login_token":"t"}}`, func() json.Unmarshaler { return &AuthorizeAction{} }},

		{"PartialEntity newtype", `{"Server":{"server_id":"1","name":"n","avatar":"a","total_members":0,"online_members":0,"short":"s","type":"t","votes":0,"invite_clicks":0,"clicks":0,"nsfw":false,"tags":[],"premium":false,"claimed_by":null,"last_claimed":null,"mentionable":[]}}`, func() json.Unmarshaler { return &PartialEntity{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.into()

			if err := json.Unmarshal([]byte(tt.wire), target); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			out, err := json.Marshal(target)

			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			if string(out) != tt.wire {
				t.Errorf("round-trip mismatch\n got: %s\nwant: %s", out, tt.wire)
			}
		})
	}
}

// Every one of the 18 RPC methods must round-trip, since rpc_logs.data stores
// exactly this encoding.
func TestRPCMethodsRoundTrip(t *testing.T) {
	wire := map[string]string{
		"Claim":                    `{"Claim":{"target_id":"1","force":true}}`,
		"Unclaim":                  `{"Unclaim":{"target_id":"1","reason":"r"}}`,
		"Approve":                  `{"Approve":{"target_id":"1","reason":"r"}}`,
		"Deny":                     `{"Deny":{"target_id":"1","reason":"r"}}`,
		"Unverify":                 `{"Unverify":{"target_id":"1","reason":"r"}}`,
		"PremiumAdd":               `{"PremiumAdd":{"target_id":"1","reason":"r","time_period_hours":24}}`,
		"PremiumRemove":            `{"PremiumRemove":{"target_id":"1","reason":"r"}}`,
		"VoteBanAdd":               `{"VoteBanAdd":{"target_id":"1","reason":"r"}}`,
		"VoteBanRemove":            `{"VoteBanRemove":{"target_id":"1","reason":"r"}}`,
		"VoteReset":                `{"VoteReset":{"target_id":"1","reason":"r"}}`,
		"VoteResetAll":             `{"VoteResetAll":{"reason":"r"}}`,
		"ForceRemove":              `{"ForceRemove":{"target_id":"1","reason":"r","kick":false}}`,
		"CertifyAdd":               `{"CertifyAdd":{"target_id":"1","reason":"r"}}`,
		"CertifyRemove":            `{"CertifyRemove":{"target_id":"1","reason":"r"}}`,
		"BotTransferOwnershipUser": `{"BotTransferOwnershipUser":{"target_id":"1","reason":"r","new_owner":"2"}}`,
		"BotTransferOwnershipTeam": `{"BotTransferOwnershipTeam":{"target_id":"1","reason":"r","new_team":"2"}}`,
		"AppBanUser":               `{"AppBanUser":{"target_id":"1","reason":"r"}}`,
		"AppUnbanUser":             `{"AppUnbanUser":{"target_id":"1","reason":"r"}}`,
	}

	if len(wire) != len(RPCMethodVariants) {
		t.Fatalf("test covers %d methods, there are %d", len(wire), len(RPCMethodVariants))
	}

	for _, name := range RPCMethodVariants {
		t.Run(name, func(t *testing.T) {
			var method RPCMethod

			if err := json.Unmarshal([]byte(wire[name]), &method); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if method.Name() != name {
				t.Errorf("Name() = %q, want %q", method.Name(), name)
			}

			out, err := json.Marshal(method)

			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			if string(out) != wire[name] {
				t.Errorf("round-trip mismatch\n got: %s\nwant: %s", out, wire[name])
			}
		})
	}
}

// A struct variant sent as a bare string, and a unit variant sent with a
// payload, must both be rejected rather than silently accepted.
func TestUnionRejectsWrongShape(t *testing.T) {
	var action StaffPositionAction

	if err := json.Unmarshal([]byte(`"SetIndex"`), &action); err == nil {
		t.Error("expected an error decoding a struct variant as a bare string")
	}

	if err := json.Unmarshal([]byte(`{"ListPositions":{"x":1}}`), &action); err == nil {
		t.Error("expected an error decoding a unit variant with an object payload")
	}

	if err := json.Unmarshal([]byte(`{"NotAVariant":{}}`), &action); err == nil {
		t.Error("expected an error decoding an unknown variant")
	}

	// A two-key object is not a valid externally tagged enum.
	if err := json.Unmarshal([]byte(`{"ListPositions":null,"SetIndex":{}}`), &action); err == nil {
		t.Error("expected an error decoding a multi-key object")
	}
}

// serde emits null for None and [] for an empty Vec. omitempty anywhere in these
// DTOs would break the panel's parsing.
func TestNullAndEmptyEncoding(t *testing.T) {
	bot := PartialBot{Mentionable: []string{}}

	out, err := json.Marshal(bot)

	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for _, want := range []string{`"claimed_by":null`, `"last_claimed":null`, `"mentionable":[]`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("encoding is missing %s\ngot: %s", want, out)
		}
	}
}

// StaffMember must carry exactly one resolved_perms key and no staff_permission
// key (§3.3).
func TestStaffMemberSerialization(t *testing.T) {
	member := StaffMember{
		UserID:        "1",
		Positions:     []StaffPosition{},
		PermOverrides: []string{},
		ResolvedPerms: []string{"rpc.Claim"},
	}

	out, err := json.Marshal(member)

	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if got := strings.Count(string(out), `"resolved_perms"`); got != 1 {
		t.Errorf("resolved_perms appears %d times, want 1\ngot: %s", got, out)
	}

	if strings.Contains(string(out), "staff_permission") {
		t.Errorf("staff_permission must not appear on the wire\ngot: %s", out)
	}
}

// The RPC form metadata drives both the panel UI and the Discord modal, so every
// method must expose fields, a label and a description.
func TestRPCMetadataIsComplete(t *testing.T) {
	for _, name := range RPCMethodVariants {
		method, err := EmptyRPCMethod(name)

		if err != nil {
			t.Fatalf("EmptyRPCMethod(%q): %v", name, err)
		}

		if method.Label() == "" {
			t.Errorf("%s has no label", name)
		}

		if method.Description() == "" {
			t.Errorf("%s has no description", name)
		}

		if len(method.Fields()) == 0 {
			t.Errorf("%s has no fields", name)
		}

		if len(method.SupportedTargetTypes()) == 0 {
			t.Errorf("%s supports no target types", name)
		}
	}
}
