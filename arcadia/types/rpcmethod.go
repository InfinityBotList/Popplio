package types

import (
	"encoding/json"
	"fmt"
)

// MaxReasonLength caps every `reason` field on an RPC method.
const MaxReasonLength = 2000

// RPCMethod is the tagged union of the 18 staff actions. Exactly one field is
// non-nil.
//
// Its Display form (Name) is load-bearing in three places:
//
//	the permission string  "rpc." + Name  ->  "rpc.Claim"
//	the `method` column in rpc_logs
//	the leaderboard query, which filters method IN ('Approve','Deny')
type RPCMethod struct {
	Claim                    *RPCClaim
	Unclaim                  *RPCTargetReason
	Approve                  *RPCTargetReason
	Deny                     *RPCTargetReason
	Unverify                 *RPCTargetReason
	PremiumAdd               *RPCPremiumAdd
	PremiumRemove            *RPCTargetReason
	VoteBanAdd               *RPCTargetReason
	VoteBanRemove            *RPCTargetReason
	VoteReset                *RPCTargetReason
	VoteResetAll             *RPCVoteResetAll
	ForceRemove              *RPCForceRemove
	CertifyAdd               *RPCTargetReason
	CertifyRemove            *RPCTargetReason
	BotTransferOwnershipUser *RPCBotTransferOwnershipUser
	BotTransferOwnershipTeam *RPCBotTransferOwnershipTeam
	AppBanUser               *RPCTargetReason
	AppUnbanUser             *RPCTargetReason
}

// RPCTargetReason is the shape shared by every method that takes just a target
// and a reason. Field order matches the Rust declaration order, which is what
// ends up in the rpc_logs.data column.
type RPCTargetReason struct {
	TargetID string `json:"target_id"`
	Reason   string `json:"reason"`
}

type RPCClaim struct {
	TargetID string `json:"target_id"`
	Force    bool   `json:"force"`
}

type RPCPremiumAdd struct {
	TargetID        string `json:"target_id"`
	Reason          string `json:"reason"`
	TimePeriodHours int32  `json:"time_period_hours"`
}

type RPCVoteResetAll struct {
	Reason string `json:"reason"`
}

type RPCForceRemove struct {
	TargetID string `json:"target_id"`
	Reason   string `json:"reason"`
	Kick     bool   `json:"kick"`
}

type RPCBotTransferOwnershipUser struct {
	TargetID string `json:"target_id"`
	Reason   string `json:"reason"`
	NewOwner string `json:"new_owner"`
}

type RPCBotTransferOwnershipTeam struct {
	TargetID string `json:"target_id"`
	Reason   string `json:"reason"`
	NewTeam  string `json:"new_team"`
}

// RPCMethodVariants is in Rust declaration order. GetRpcMethods and the Discord
// `rpclist` command both iterate it, so the order is user-visible.
var RPCMethodVariants = []string{
	"Claim",
	"Unclaim",
	"Approve",
	"Deny",
	"Unverify",
	"PremiumAdd",
	"PremiumRemove",
	"VoteBanAdd",
	"VoteBanRemove",
	"VoteReset",
	"VoteResetAll",
	"ForceRemove",
	"CertifyAdd",
	"CertifyRemove",
	"BotTransferOwnershipUser",
	"BotTransferOwnershipTeam",
	"AppBanUser",
	"AppUnbanUser",
}

// Name is the Display impl: the variant name.
func (m RPCMethod) Name() string {
	switch {
	case m.Claim != nil:
		return "Claim"
	case m.Unclaim != nil:
		return "Unclaim"
	case m.Approve != nil:
		return "Approve"
	case m.Deny != nil:
		return "Deny"
	case m.Unverify != nil:
		return "Unverify"
	case m.PremiumAdd != nil:
		return "PremiumAdd"
	case m.PremiumRemove != nil:
		return "PremiumRemove"
	case m.VoteBanAdd != nil:
		return "VoteBanAdd"
	case m.VoteBanRemove != nil:
		return "VoteBanRemove"
	case m.VoteReset != nil:
		return "VoteReset"
	case m.VoteResetAll != nil:
		return "VoteResetAll"
	case m.ForceRemove != nil:
		return "ForceRemove"
	case m.CertifyAdd != nil:
		return "CertifyAdd"
	case m.CertifyRemove != nil:
		return "CertifyRemove"
	case m.BotTransferOwnershipUser != nil:
		return "BotTransferOwnershipUser"
	case m.BotTransferOwnershipTeam != nil:
		return "BotTransferOwnershipTeam"
	case m.AppBanUser != nil:
		return "AppBanUser"
	case m.AppUnbanUser != nil:
		return "AppUnbanUser"
	default:
		return ""
	}
}

func (m RPCMethod) String() string {
	return m.Name()
}

// EmptyRPCMethod builds a zero-valued method of the named variant. The panel's
// GetRpcMethods listing and the Discord modal driver both need one instance per
// variant to read its metadata off.
func EmptyRPCMethod(name string) (RPCMethod, error) {
	var m RPCMethod

	switch name {
	case "Claim":
		m.Claim = &RPCClaim{}
	case "Unclaim":
		m.Unclaim = &RPCTargetReason{}
	case "Approve":
		m.Approve = &RPCTargetReason{}
	case "Deny":
		m.Deny = &RPCTargetReason{}
	case "Unverify":
		m.Unverify = &RPCTargetReason{}
	case "PremiumAdd":
		m.PremiumAdd = &RPCPremiumAdd{}
	case "PremiumRemove":
		m.PremiumRemove = &RPCTargetReason{}
	case "VoteBanAdd":
		m.VoteBanAdd = &RPCTargetReason{}
	case "VoteBanRemove":
		m.VoteBanRemove = &RPCTargetReason{}
	case "VoteReset":
		m.VoteReset = &RPCTargetReason{}
	case "VoteResetAll":
		m.VoteResetAll = &RPCVoteResetAll{}
	case "ForceRemove":
		m.ForceRemove = &RPCForceRemove{}
	case "CertifyAdd":
		m.CertifyAdd = &RPCTargetReason{}
	case "CertifyRemove":
		m.CertifyRemove = &RPCTargetReason{}
	case "BotTransferOwnershipUser":
		m.BotTransferOwnershipUser = &RPCBotTransferOwnershipUser{}
	case "BotTransferOwnershipTeam":
		m.BotTransferOwnershipTeam = &RPCBotTransferOwnershipTeam{}
	case "AppBanUser":
		m.AppBanUser = &RPCTargetReason{}
	case "AppUnbanUser":
		m.AppUnbanUser = &RPCTargetReason{}
	default:
		return m, errUnknownVariant("RPCMethod", name)
	}

	return m, nil
}

func (m *RPCMethod) UnmarshalJSON(data []byte) error {
	variant, payload, err := decodeUnion(data)
	if err != nil {
		return fmt.Errorf("RPCMethod: %w", err)
	}

	if payload == nil {
		// Every RPCMethod variant is a struct variant, so a bare string is never
		// valid here.
		return errVariantNeedsPayload("RPCMethod", variant)
	}

	built, err := EmptyRPCMethod(variant)
	if err != nil {
		return err
	}

	*m = built

	switch variant {
	case "Claim":
		return json.Unmarshal(payload, m.Claim)
	case "Unclaim":
		return json.Unmarshal(payload, m.Unclaim)
	case "Approve":
		return json.Unmarshal(payload, m.Approve)
	case "Deny":
		return json.Unmarshal(payload, m.Deny)
	case "Unverify":
		return json.Unmarshal(payload, m.Unverify)
	case "PremiumAdd":
		return json.Unmarshal(payload, m.PremiumAdd)
	case "PremiumRemove":
		return json.Unmarshal(payload, m.PremiumRemove)
	case "VoteBanAdd":
		return json.Unmarshal(payload, m.VoteBanAdd)
	case "VoteBanRemove":
		return json.Unmarshal(payload, m.VoteBanRemove)
	case "VoteReset":
		return json.Unmarshal(payload, m.VoteReset)
	case "VoteResetAll":
		return json.Unmarshal(payload, m.VoteResetAll)
	case "ForceRemove":
		return json.Unmarshal(payload, m.ForceRemove)
	case "CertifyAdd":
		return json.Unmarshal(payload, m.CertifyAdd)
	case "CertifyRemove":
		return json.Unmarshal(payload, m.CertifyRemove)
	case "BotTransferOwnershipUser":
		return json.Unmarshal(payload, m.BotTransferOwnershipUser)
	case "BotTransferOwnershipTeam":
		return json.Unmarshal(payload, m.BotTransferOwnershipTeam)
	case "AppBanUser":
		return json.Unmarshal(payload, m.AppBanUser)
	case "AppUnbanUser":
		return json.Unmarshal(payload, m.AppUnbanUser)
	}

	return errUnknownVariant("RPCMethod", variant)
}

func (m RPCMethod) MarshalJSON() ([]byte, error) {
	name := m.Name()
	if name == "" {
		return nil, fmt.Errorf("RPCMethod: no variant set")
	}

	return encodeVariant(name, m.inner())
}

func (m RPCMethod) inner() any {
	switch {
	case m.Claim != nil:
		return m.Claim
	case m.Unclaim != nil:
		return m.Unclaim
	case m.Approve != nil:
		return m.Approve
	case m.Deny != nil:
		return m.Deny
	case m.Unverify != nil:
		return m.Unverify
	case m.PremiumAdd != nil:
		return m.PremiumAdd
	case m.PremiumRemove != nil:
		return m.PremiumRemove
	case m.VoteBanAdd != nil:
		return m.VoteBanAdd
	case m.VoteBanRemove != nil:
		return m.VoteBanRemove
	case m.VoteReset != nil:
		return m.VoteReset
	case m.VoteResetAll != nil:
		return m.VoteResetAll
	case m.ForceRemove != nil:
		return m.ForceRemove
	case m.CertifyAdd != nil:
		return m.CertifyAdd
	case m.CertifyRemove != nil:
		return m.CertifyRemove
	case m.BotTransferOwnershipUser != nil:
		return m.BotTransferOwnershipUser
	case m.BotTransferOwnershipTeam != nil:
		return m.BotTransferOwnershipTeam
	case m.AppBanUser != nil:
		return m.AppBanUser
	case m.AppUnbanUser != nil:
		return m.AppUnbanUser
	default:
		return nil
	}
}

// SupportedTargetTypes lists the target types the method accepts.
func (m RPCMethod) SupportedTargetTypes() []TargetType {
	switch m.Name() {
	case "VoteReset", "VoteResetAll":
		return []TargetType{TargetTypeBot, TargetTypeServer, TargetTypeTeam, TargetTypePack}
	case "AppBanUser", "AppUnbanUser":
		return []TargetType{TargetTypeUser}
	case "":
		return []TargetType{}
	default:
		return []TargetType{TargetTypeBot}
	}
}

// Description is user-visible in the panel and in `rpclist`. Frozen.
func (m RPCMethod) Description() string {
	switch m.Name() {
	case "Claim":
		return "Claim a entity. Be sure to claim entities that you are going to review!"
	case "Unclaim":
		return "Unclaim a entity. Be sure to use this if you can't review the entity!"
	case "Approve":
		return "Approve a entity. Needs to be claimed first."
	case "Deny":
		return "Deny a entity. Needs to be claimed first."
	case "Unverify":
		return "Unverifies a bot on the list"
	case "PremiumAdd":
		return "Adds premium to a bot for a given time period"
	case "PremiumRemove":
		return "Removes premium from a bot"
	case "VoteBanAdd":
		return "Vote-bans the bot in question"
	case "VoteBanRemove":
		return "Removes the vote-ban from the bot in question"
	case "VoteReset":
		return "Reset the votes of a given entity (bot/pack/server etc."
	case "VoteResetAll":
		return "Reset the votes of a given entity (bot/pack/server etc."
	case "ForceRemove":
		return "Forcefully removes a bot from the list"
	case "CertifyAdd":
		return "Certifies a entity. Recommended to use apps instead however"
	case "CertifyRemove":
		return "Uncertifies a bot"
	case "BotTransferOwnershipUser":
		return "Transfers the ownership of a bot to a new user"
	case "BotTransferOwnershipTeam":
		return "Transfers the ownership of a bot to a new team"
	case "AppBanUser":
		return "Ban user from apps"
	case "AppUnbanUser":
		return "Unban user from apps"
	default:
		return ""
	}
}

// Label is user-visible in the panel and in `rpclist`. Frozen.
func (m RPCMethod) Label() string {
	switch m.Name() {
	case "Claim":
		return "Claim entity"
	case "Unclaim":
		return "Unclaim entity"
	case "Approve":
		return "Approve entity"
	case "Deny":
		return "Deny entity"
	case "Unverify":
		return "Unverify entity"
	case "PremiumAdd":
		return "Add Premium"
	case "PremiumRemove":
		return "Remove Premium"
	case "VoteBanAdd":
		return "Vote Ban"
	case "VoteBanRemove":
		return "Unvote Ban"
	case "VoteReset":
		return "Vote Reset Entity"
	case "VoteResetAll":
		return "Vote Reset All Entities"
	case "ForceRemove":
		return "Force Remove"
	case "CertifyAdd":
		return "Certify"
	case "CertifyRemove":
		return "Uncertify"
	case "BotTransferOwnershipUser":
		return "Set Bot Owner [User]"
	case "BotTransferOwnershipTeam":
		return "Set Bot Owner [Team]"
	case "AppBanUser":
		return "Ban from apps [User]"
	case "AppUnbanUser":
		return "Unban from apps [User]"
	default:
		return ""
	}
}

// FieldType drives both the panel form renderer and the Discord modal driver.
type FieldType string

const (
	FieldTypeText     FieldType = "Text"
	FieldTypeTextarea FieldType = "Textarea"
	FieldTypeNumber   FieldType = "Number"
	FieldTypeHour     FieldType = "Hour"
	FieldTypeBoolean  FieldType = "Boolean"
)

// RPCField is a single form field of an RPC method.
type RPCField struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	FieldType   FieldType `json:"field_type"`
	Icon        string    `json:"icon"`
	Placeholder string    `json:"placeholder"`
}

func rpcFieldTargetID() RPCField {
	return RPCField{
		ID:          "target_id",
		Label:       "Target ID",
		FieldType:   FieldTypeText,
		Icon:        "ic:twotone-access-time-filled",
		Placeholder: "The Target ID to perform the action on",
	}
}

func rpcFieldReason() RPCField {
	return RPCField{
		ID:          "reason",
		Label:       "Reason",
		FieldType:   FieldTypeTextarea,
		Icon:        "material-symbols:question-mark",
		Placeholder: "Reason for performing this action",
	}
}

// Fields returns the form fields of the method. Labels and placeholders are
// user-visible and frozen.
func (m RPCMethod) Fields() []RPCField {
	switch m.Name() {
	case "Claim":
		return []RPCField{
			rpcFieldTargetID(),
			{
				ID:          "force",
				Label:       "Force claim bot?",
				FieldType:   FieldTypeBoolean,
				Icon:        "fa-solid:sign-out-alt",
				Placeholder: "Yes/No",
			},
		}
	case "PremiumAdd":
		return []RPCField{
			rpcFieldTargetID(),
			{
				ID:          "time_period_hours",
				Label:       "Time [X unit(s)]",
				FieldType:   FieldTypeHour,
				Icon:        "material-symbols:timer",
				Placeholder: "Time period. Format: X years/days/hours",
			},
			rpcFieldReason(),
		}
	case "VoteResetAll":
		return []RPCField{rpcFieldReason()}
	case "ForceRemove":
		return []RPCField{
			rpcFieldTargetID(),
			{
				ID:          "kick",
				Label:       "Kick the bot from the server",
				FieldType:   FieldTypeBoolean,
				Icon:        "fa-solid:sign-out-alt",
				Placeholder: "Kick the bot from the server",
			},
			rpcFieldReason(),
		}
	case "BotTransferOwnershipUser":
		return []RPCField{
			rpcFieldTargetID(),
			{
				ID:          "new_owner",
				Label:       "User ID",
				FieldType:   FieldTypeText,
				Icon:        "material-symbols:timer",
				Placeholder: "New Owner",
			},
			rpcFieldReason(),
		}
	case "BotTransferOwnershipTeam":
		return []RPCField{
			rpcFieldTargetID(),
			{
				ID:          "new_team",
				Label:       "Team ID",
				FieldType:   FieldTypeText,
				Icon:        "material-symbols:timer",
				Placeholder: "New Team",
			},
			rpcFieldReason(),
		}
	case "":
		return []RPCField{}
	default:
		return []RPCField{rpcFieldTargetID(), rpcFieldReason()}
	}
}

// RPCWebAction is one entry of the GetRpcMethods response. Field order matches
// the Rust declaration.
type RPCWebAction struct {
	ID                   string       `json:"id"`
	Label                string       `json:"label"`
	Description          string       `json:"description"`
	Fields               []RPCField   `json:"fields"`
	SupportedTargetTypes []TargetType `json:"supported_target_types"`
}
