package types

import (
	"fmt"

	perms "github.com/infinitybotlist/kittycat/go"
)

// StaffPositionAction is the union of staff position operations.
type StaffPositionAction struct {
	ListPositions  *Unit
	SwapIndex      *StaffSwapIndex
	SetIndex       *StaffSetIndex
	CreatePosition *StaffCreatePosition
	EditPosition   *StaffEditPosition
	DeletePosition *StaffDeletePosition
}

type StaffSwapIndex struct {
	A string `json:"a"`
	B string `json:"b"`
}

type StaffSetIndex struct {
	ID    string `json:"id"`
	Index int32  `json:"index"`
}

type StaffCreatePosition struct {
	Name               string   `json:"name"`
	RoleID             string   `json:"role_id"`
	CorrespondingRoles []Link   `json:"corresponding_roles"`
	Perms              []string `json:"perms"`
	Icon               string   `json:"icon"`
	Index              int32    `json:"index"`
}

type StaffEditPosition struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	RoleID             string   `json:"role_id"`
	CorrespondingRoles []Link   `json:"corresponding_roles"`
	Perms              []string `json:"perms"`
	Icon               string   `json:"icon"`
}

type StaffDeletePosition struct {
	ID string `json:"id"`
}

func (a *StaffPositionAction) UnmarshalJSON(data []byte) error {
	*a = StaffPositionAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("StaffPositionAction: %w", err)
	}

	switch name {
	case "ListPositions":
		a.ListPositions = unitSet()
	case "SwapIndex":
		a.SwapIndex = &StaffSwapIndex{}
		return decodeVariant("StaffPositionAction", name, payload, a.SwapIndex)
	case "SetIndex":
		a.SetIndex = &StaffSetIndex{}
		return decodeVariant("StaffPositionAction", name, payload, a.SetIndex)
	case "CreatePosition":
		a.CreatePosition = &StaffCreatePosition{}
		return decodeVariant("StaffPositionAction", name, payload, a.CreatePosition)
	case "EditPosition":
		a.EditPosition = &StaffEditPosition{}
		return decodeVariant("StaffPositionAction", name, payload, a.EditPosition)
	case "DeletePosition":
		a.DeletePosition = &StaffDeletePosition{}
		return decodeVariant("StaffPositionAction", name, payload, a.DeletePosition)
	default:
		return errUnknownVariant("StaffPositionAction", name)
	}

	return expectUnit("StaffPositionAction", name, payload)
}

func (a StaffPositionAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListPositions != nil:
		return encodeUnit("ListPositions")
	case a.SwapIndex != nil:
		return encodeVariant("SwapIndex", a.SwapIndex)
	case a.SetIndex != nil:
		return encodeVariant("SetIndex", a.SetIndex)
	case a.CreatePosition != nil:
		return encodeVariant("CreatePosition", a.CreatePosition)
	case a.EditPosition != nil:
		return encodeVariant("EditPosition", a.EditPosition)
	case a.DeletePosition != nil:
		return encodeVariant("DeletePosition", a.DeletePosition)
	default:
		return nil, fmt.Errorf("StaffPositionAction: no variant set")
	}
}

type StaffPosition struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	RoleID             string    `json:"role_id"`
	Perms              []string  `json:"perms"`
	CorrespondingRoles []Link    `json:"corresponding_roles"`
	Icon               string    `json:"icon"`
	Index              int32     `json:"index"`
	CreatedAt          Timestamp `json:"created_at"`
}

// CorrespondingServer names the guild a position's corresponding role lives in.
type CorrespondingServer string

const (
	CorrespondingServerMain    CorrespondingServer = "Main"
	CorrespondingServerTesting CorrespondingServer = "Testing"
	CorrespondingServerStaff   CorrespondingServer = "Staff"
)

// CorrespondingServerFromString mirrors the Rust FromStr, which matches on the
// LOWERCASE link name, not the variant name.
func CorrespondingServerFromString(s string) (CorrespondingServer, error) {
	switch s {
	case "main":
		return CorrespondingServerMain, nil
	case "testing":
		return CorrespondingServerTesting, nil
	case "staff":
		return CorrespondingServerStaff, nil
	default:
		return "", fmt.Errorf("Invalid corresponding server: %s", s)
	}
}

// CorrespondingServerNames is the list quoted back at the user when a link name
// does not resolve. Order matches strum's VARIANTS (declaration order).
var CorrespondingServerNames = []string{"Main", "Testing", "Staff"}

// StaffMemberAction is the union of staff member operations.
type StaffMemberAction struct {
	ListMembers *Unit
	EditMember  *StaffEditMember
}

type StaffEditMember struct {
	UserID        string   `json:"user_id"`
	PermOverrides []string `json:"perm_overrides"`
	NoAutosync    bool     `json:"no_autosync"`
	Unaccounted   bool     `json:"unaccounted"`
}

func (a *StaffMemberAction) UnmarshalJSON(data []byte) error {
	*a = StaffMemberAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("StaffMemberAction: %w", err)
	}

	switch name {
	case "ListMembers":
		a.ListMembers = unitSet()
	case "EditMember":
		a.EditMember = &StaffEditMember{}
		return decodeVariant("StaffMemberAction", name, payload, a.EditMember)
	default:
		return errUnknownVariant("StaffMemberAction", name)
	}

	return expectUnit("StaffMemberAction", name, payload)
}

func (a StaffMemberAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListMembers != nil:
		return encodeUnit("ListMembers")
	case a.EditMember != nil:
		return encodeVariant("EditMember", a.EditMember)
	default:
		return nil, fmt.Errorf("StaffMemberAction: no variant set")
	}
}

// StaffMember is the fully resolved staff member.
//
// Serialization note (§3.3): the Rust struct carries both `resolved_perms`
// (skipped) and `resolved_perms_kc` (renamed to "resolved_perms"), plus a
// skipped `staff_permission`. The net wire shape is exactly one "resolved_perms"
// key holding an array of permission strings, and no "staff_permission" key.
type StaffMember struct {
	UserID         string              `json:"user_id"`
	User           PlatformUser        `json:"user"`
	Positions      []StaffPosition     `json:"positions"`
	Disciplinaries []StaffDisciplinary `json:"disciplinaries"`
	PermOverrides  []string            `json:"perm_overrides"`
	ResolvedPerms  []string            `json:"resolved_perms"`
	NoAutosync     bool                `json:"no_autosync"`
	Unaccounted    bool                `json:"unaccounted"`
	MfaVerified    bool                `json:"mfa_verified"`
	CreatedAt      Timestamp           `json:"created_at"`

	// StaffPermission is the pre-disciplinary permission set. It is skipped by
	// serde and must never appear on the wire, hence json:"-".
	StaffPermission perms.StaffPermissions `json:"-"`
}

// StaffDisciplinaryTypeAction is the union of disciplinary type operations.
type StaffDisciplinaryTypeAction struct {
	ListDisciplinaryTypes  *Unit
	CreateDisciplinaryType *StaffDisciplinaryTypeUpsert
	EditDisciplinaryType   *StaffDisciplinaryTypeUpsert
	DeleteDisciplinaryType *StaffDisciplinaryTypeDelete
}

type StaffDisciplinaryTypeUpsert struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	SelfAssignable bool     `json:"self_assignable"`
	PermLimits     []string `json:"perm_limits"`
	Additory       bool     `json:"additory"`
	NeedsApproval  bool     `json:"needs_approval"`
	MaxExpiry      *float64 `json:"max_expiry"`
}

type StaffDisciplinaryTypeDelete struct {
	ID string `json:"id"`
}

func (a *StaffDisciplinaryTypeAction) UnmarshalJSON(data []byte) error {
	*a = StaffDisciplinaryTypeAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("StaffDisciplinaryTypeAction: %w", err)
	}

	switch name {
	case "ListDisciplinaryTypes":
		a.ListDisciplinaryTypes = unitSet()
	case "CreateDisciplinaryType":
		a.CreateDisciplinaryType = &StaffDisciplinaryTypeUpsert{}
		return decodeVariant("StaffDisciplinaryTypeAction", name, payload, a.CreateDisciplinaryType)
	case "EditDisciplinaryType":
		a.EditDisciplinaryType = &StaffDisciplinaryTypeUpsert{}
		return decodeVariant("StaffDisciplinaryTypeAction", name, payload, a.EditDisciplinaryType)
	case "DeleteDisciplinaryType":
		a.DeleteDisciplinaryType = &StaffDisciplinaryTypeDelete{}
		return decodeVariant("StaffDisciplinaryTypeAction", name, payload, a.DeleteDisciplinaryType)
	default:
		return errUnknownVariant("StaffDisciplinaryTypeAction", name)
	}

	return expectUnit("StaffDisciplinaryTypeAction", name, payload)
}

func (a StaffDisciplinaryTypeAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListDisciplinaryTypes != nil:
		return encodeUnit("ListDisciplinaryTypes")
	case a.CreateDisciplinaryType != nil:
		return encodeVariant("CreateDisciplinaryType", a.CreateDisciplinaryType)
	case a.EditDisciplinaryType != nil:
		return encodeVariant("EditDisciplinaryType", a.EditDisciplinaryType)
	case a.DeleteDisciplinaryType != nil:
		return encodeVariant("DeleteDisciplinaryType", a.DeleteDisciplinaryType)
	default:
		return nil, fmt.Errorf("StaffDisciplinaryTypeAction: no variant set")
	}
}

type StaffDisciplinaryType struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	SelfAssignable bool      `json:"self_assignable"`
	PermLimits     []string  `json:"perm_limits"`
	Additory       bool      `json:"additory"`
	NeedsApproval  bool      `json:"needs_approval"`
	MaxExpiry      *float64  `json:"max_expiry"`
	CreatedAt      Timestamp `json:"created_at"`
}

type StaffDisciplinary struct {
	ID          string                `json:"id"`
	UserID      string                `json:"user_id"`
	CreatedAt   Timestamp             `json:"created_at"`
	ExpiresAt   *int64                `json:"expires_at"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Type        StaffDisciplinaryType `json:"type"`
}
