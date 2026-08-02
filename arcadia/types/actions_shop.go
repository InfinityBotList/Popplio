package types

import "fmt"

// VoteCreditTierAction is the union of vote credit tier operations.
type VoteCreditTierAction struct {
	ListTiers  *Unit
	CreateTier *VoteCreditTierUpsert
	EditTier   *VoteCreditTierUpsert
	DeleteTier *VoteCreditTierDelete
}

type VoteCreditTierUpsert struct {
	ID         string  `json:"id"`
	TargetType string  `json:"target_type"`
	Position   int32   `json:"position"`
	Cents      float64 `json:"cents"`
	Votes      int32   `json:"votes"`
}

type VoteCreditTierDelete struct {
	ID string `json:"id"`
}

func (a *VoteCreditTierAction) UnmarshalJSON(data []byte) error {
	*a = VoteCreditTierAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("VoteCreditTierAction: %w", err)
	}

	switch name {
	case "ListTiers":
		a.ListTiers = unitSet()
	case "CreateTier":
		a.CreateTier = &VoteCreditTierUpsert{}
		return decodeVariant("VoteCreditTierAction", name, payload, a.CreateTier)
	case "EditTier":
		a.EditTier = &VoteCreditTierUpsert{}
		return decodeVariant("VoteCreditTierAction", name, payload, a.EditTier)
	case "DeleteTier":
		a.DeleteTier = &VoteCreditTierDelete{}
		return decodeVariant("VoteCreditTierAction", name, payload, a.DeleteTier)
	default:
		return errUnknownVariant("VoteCreditTierAction", name)
	}

	return expectUnit("VoteCreditTierAction", name, payload)
}

func (a VoteCreditTierAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListTiers != nil:
		return encodeUnit("ListTiers")
	case a.CreateTier != nil:
		return encodeVariant("CreateTier", a.CreateTier)
	case a.EditTier != nil:
		return encodeVariant("EditTier", a.EditTier)
	case a.DeleteTier != nil:
		return encodeVariant("DeleteTier", a.DeleteTier)
	default:
		return nil, fmt.Errorf("VoteCreditTierAction: no variant set")
	}
}

type VoteCreditTier struct {
	ID         string    `json:"id"`
	TargetType string    `json:"target_type"`
	Position   int32     `json:"position"`
	Cents      float64   `json:"cents"`
	Votes      int32     `json:"votes"`
	CreatedAt  Timestamp `json:"created_at"`
}

// ShopItemAction is the union of shop item operations.
type ShopItemAction struct {
	List   *Unit
	Create *ShopItemUpsert
	Edit   *ShopItemUpsert
	Delete *ShopItemDelete
}

type ShopItemUpsert struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Cents       float64  `json:"cents"`
	TargetTypes []string `json:"target_types"`
	Benefits    []string `json:"benefits"`
	Duration    int32    `json:"duration"`
}

type ShopItemDelete struct {
	ID string `json:"id"`
}

func (a *ShopItemAction) UnmarshalJSON(data []byte) error {
	*a = ShopItemAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("ShopItemAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "Create":
		a.Create = &ShopItemUpsert{}
		return decodeVariant("ShopItemAction", name, payload, a.Create)
	case "Edit":
		a.Edit = &ShopItemUpsert{}
		return decodeVariant("ShopItemAction", name, payload, a.Edit)
	case "Delete":
		a.Delete = &ShopItemDelete{}
		return decodeVariant("ShopItemAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("ShopItemAction", name)
	}

	return expectUnit("ShopItemAction", name, payload)
}

func (a ShopItemAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Edit != nil:
		return encodeVariant("Edit", a.Edit)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("ShopItemAction: no variant set")
	}
}

type ShopItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Cents       float64   `json:"cents"`
	TargetTypes []string  `json:"target_types"`
	Benefits    []string  `json:"benefits"`
	Duration    int32     `json:"duration"`
	CreatedAt   Timestamp `json:"created_at"`
	LastUpdated Timestamp `json:"last_updated"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
}

// ShopItemBenefitAction is the union of shop item benefit operations.
type ShopItemBenefitAction struct {
	List   *Unit
	Create *ShopItemBenefitUpsert
	Edit   *ShopItemBenefitUpsert
	Delete *ShopItemBenefitDelete
}

type ShopItemBenefitUpsert struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TargetTypes []string `json:"target_types"`
}

type ShopItemBenefitDelete struct {
	ID string `json:"id"`
}

func (a *ShopItemBenefitAction) UnmarshalJSON(data []byte) error {
	*a = ShopItemBenefitAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("ShopItemBenefitAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "Create":
		a.Create = &ShopItemBenefitUpsert{}
		return decodeVariant("ShopItemBenefitAction", name, payload, a.Create)
	case "Edit":
		a.Edit = &ShopItemBenefitUpsert{}
		return decodeVariant("ShopItemBenefitAction", name, payload, a.Edit)
	case "Delete":
		a.Delete = &ShopItemBenefitDelete{}
		return decodeVariant("ShopItemBenefitAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("ShopItemBenefitAction", name)
	}

	return expectUnit("ShopItemBenefitAction", name, payload)
}

func (a ShopItemBenefitAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Edit != nil:
		return encodeVariant("Edit", a.Edit)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("ShopItemBenefitAction: no variant set")
	}
}

type ShopItemBenefit struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   Timestamp `json:"created_at"`
	LastUpdated Timestamp `json:"last_updated"`
	TargetTypes []string  `json:"target_types"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
}

// ShopCouponAction is the union of shop coupon operations.
type ShopCouponAction struct {
	List   *Unit
	Create *ShopCouponUpsert
	Edit   *ShopCouponUpsert
	Delete *ShopCouponDelete
}

type ShopCouponUpsert struct {
	ID                string   `json:"id"`
	Code              string   `json:"code"`
	Public            bool     `json:"public"`
	MaxUses           *int32   `json:"max_uses"`
	ReuseWaitDuration *int32   `json:"reuse_wait_duration"`
	Expiry            *int32   `json:"expiry"`
	ApplicableItems   []string `json:"applicable_items"`
	Cents             *float64 `json:"cents"`
	Requirements      []string `json:"requirements"`
	AllowedUsers      []string `json:"allowed_users"`
	Usable            bool     `json:"usable"`
	TargetTypes       []string `json:"target_types"`
}

type ShopCouponDelete struct {
	ID string `json:"id"`
}

func (a *ShopCouponAction) UnmarshalJSON(data []byte) error {
	*a = ShopCouponAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("ShopCouponAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "Create":
		a.Create = &ShopCouponUpsert{}
		return decodeVariant("ShopCouponAction", name, payload, a.Create)
	case "Edit":
		a.Edit = &ShopCouponUpsert{}
		return decodeVariant("ShopCouponAction", name, payload, a.Edit)
	case "Delete":
		a.Delete = &ShopCouponDelete{}
		return decodeVariant("ShopCouponAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("ShopCouponAction", name)
	}

	return expectUnit("ShopCouponAction", name, payload)
}

func (a ShopCouponAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.Create != nil:
		return encodeVariant("Create", a.Create)
	case a.Edit != nil:
		return encodeVariant("Edit", a.Edit)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("ShopCouponAction: no variant set")
	}
}

type ShopCoupon struct {
	ID                string    `json:"id"`
	Code              string    `json:"code"`
	Public            bool      `json:"public"`
	MaxUses           *int32    `json:"max_uses"`
	CreatedAt         Timestamp `json:"created_at"`
	CreatedBy         string    `json:"created_by"`
	LastUpdated       Timestamp `json:"last_updated"`
	UpdatedBy         string    `json:"updated_by"`
	ReuseWaitDuration *int32    `json:"reuse_wait_duration"`
	Expiry            *int32    `json:"expiry"`
	ApplicableItems   []string  `json:"applicable_items"`
	Cents             *float64  `json:"cents"`
	Requirements      []string  `json:"requirements"`
	AllowedUsers      []string  `json:"allowed_users"`
	Usable            bool      `json:"usable"`
	TargetTypes       []string  `json:"target_types"`
}

// BotWhitelistAction is the union of bot whitelist operations.
type BotWhitelistAction struct {
	List   *Unit
	Add    *BotWhitelistUpsert
	Edit   *BotWhitelistUpsert
	Delete *BotWhitelistDelete
}

type BotWhitelistUpsert struct {
	BotID  string `json:"bot_id"`
	Reason string `json:"reason"`
}

type BotWhitelistDelete struct {
	BotID string `json:"bot_id"`
}

func (a *BotWhitelistAction) UnmarshalJSON(data []byte) error {
	*a = BotWhitelistAction{}

	name, payload, err := decodeUnion(data)

	if err != nil {
		return fmt.Errorf("BotWhitelistAction: %w", err)
	}

	switch name {
	case "List":
		a.List = unitSet()
	case "Add":
		a.Add = &BotWhitelistUpsert{}
		return decodeVariant("BotWhitelistAction", name, payload, a.Add)
	case "Edit":
		a.Edit = &BotWhitelistUpsert{}
		return decodeVariant("BotWhitelistAction", name, payload, a.Edit)
	case "Delete":
		a.Delete = &BotWhitelistDelete{}
		return decodeVariant("BotWhitelistAction", name, payload, a.Delete)
	default:
		return errUnknownVariant("BotWhitelistAction", name)
	}

	return expectUnit("BotWhitelistAction", name, payload)
}

func (a BotWhitelistAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.List != nil:
		return encodeUnit("List")
	case a.Add != nil:
		return encodeVariant("Add", a.Add)
	case a.Edit != nil:
		return encodeVariant("Edit", a.Edit)
	case a.Delete != nil:
		return encodeVariant("Delete", a.Delete)
	default:
		return nil, fmt.Errorf("BotWhitelistAction: no variant set")
	}
}

type BotWhitelist struct {
	BotID     string    `json:"bot_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	CreatedAt Timestamp `json:"created_at"`
}
