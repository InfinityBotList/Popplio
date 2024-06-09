package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// @ci table=shop_item_benefits
//
// ShopItemBenefit represents a benefit of a shop item which can then be supported in code for certain functionality.
type ShopItemBenefit struct {
	ID          string                  `db:"id" json:"id" description:"The ID of the shop item benefit"`
	Name        string                  `db:"name" json:"name" description:"The friendly name of the shop item benefit"`
	Description string                  `db:"description" json:"description" description:"The description of the shop item benefit"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at" description:"The time the shop item benefit was created"`
	LastUpdated time.Time               `db:"last_updated" json:"last_updated" description:"The time the shop item benefit was last updated"`
	CreatedByID string                  `db:"created_by" json:"-" ci:"internal"`                                                               // This field is used to populate CreatedBy
	CreatedBy   *dovetypes.PlatformUser `db:"-" json:"created_by" description:"The user who created the shop item benefit" ci:"internal"`      // CreatedByID must be parsed internally
	UpdatedByID string                  `db:"updated_by" json:"-" ci:"internal"`                                                               // This field is used to populate UpdatedBy
	UpdatedBy   *dovetypes.PlatformUser `db:"-" json:"updated_by" description:"The user who last updated the shop item benefit" ci:"internal"` // UpdatedByID must be parsed internally
	TargetTypes []string                `db:"target_types" json:"target_types" description:"The types of entities this benefit applies to/supports"`
}

// @ci table=shop_items
//
// ShopItems represent items that can be purchased in the shop.
type ShopItem struct {
	ID          string                  `db:"id" json:"id" description:"The ID of the shop item"`
	Name        string                  `db:"name" json:"name" description:"The friendly name of the shop item"`
	Cents       float64                 `db:"cents" json:"cents" description:"The cost of the shop item in cents"`
	TargetTypes []string                `db:"target_types" json:"target_types" description:"The types of entities this item can be applied to"`
	Benefits    []string                `db:"benefits" json:"benefits" description:"The benefits of the shop item (array of ids)"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at" description:"The time the shop item benefit was created"`
	LastUpdated time.Time               `db:"last_updated" json:"last_updated" description:"The time the shop item benefit was last updated"`
	CreatedByID string                  `db:"created_by" json:"-" ci:"internal"`                                                               // This field is used to populate CreatedBy
	CreatedBy   *dovetypes.PlatformUser `db:"-" json:"created_by" description:"The user who created the shop item benefit" ci:"internal"`      // CreatedByID must be parsed internally
	UpdatedByID string                  `db:"updated_by" json:"-" ci:"internal"`                                                               // This field is used to populate UpdatedBy
	UpdatedBy   *dovetypes.PlatformUser `db:"-" json:"updated_by" description:"The user who last updated the shop item benefit" ci:"internal"` // UpdatedByID must be parsed internally
	Duration    int64                   `db:"duration" json:"duration" description:"The duration the shop item will last for in hours"`
	Description string                  `db:"description" json:"description" description:"The description of the shop item"`
}

// @ci table=shop_coupons
//
// ShopCoupon represents a coupon that can be used to get a discount/complete price removal on a shop item.
type ShopCoupon struct {
	ID                string                  `db:"id" json:"id" description:"The ID of the shop coupon"`
	Code              string                  `db:"code" json:"code" description:"The code of the shop coupon"`
	Public            bool                    `db:"public" json:"public" description:"Whether the coupon is publicly listable or not"`
	MaxUses           *int                    `db:"max_uses" json:"max_uses" description:"The maximum number of times the coupon can be used. If null, the coupon can be used infinitely"`
	CreatedAt         time.Time               `db:"created_at" json:"created_at" description:"The time the shop item benefit was created"`
	LastUpdated       time.Time               `db:"last_updated" json:"last_updated" description:"The time the shop item benefit was last updated"`
	CreatedByID       string                  `db:"created_by" json:"-" ci:"internal"`                                                               // This field is used to populate CreatedBy
	CreatedBy         *dovetypes.PlatformUser `db:"-" json:"created_by" description:"The user who created the shop item benefit" ci:"internal"`      // CreatedByID must be parsed internally
	UpdatedByID       string                  `db:"updated_by" json:"-" ci:"internal"`                                                               // This field is used to populate UpdatedBy
	UpdatedBy         *dovetypes.PlatformUser `db:"-" json:"updated_by" description:"The user who last updated the shop item benefit" ci:"internal"` // UpdatedByID must be parsed internally
	ReuseWaitDuration *int                    `db:"reuse_wait_duration" json:"reuse_wait_duration" description:"The duration in hours that must pass before the coupon can be used again. If null, the coupon has no reuse wait duration and can be immediately reapplied"`
	Expiry            *int                    `db:"expiry" json:"expiry" description:"The duration in hours that the coupon will expire after being created. If null, the coupon will never expire"`
	ApplicableItems   []string                `db:"applicable_items" json:"applicable_items" description:"The items the coupon can be applied to (array of shop item ids)"`
	Usable            bool                    `db:"usable" json:"usable" description:"Whether the coupon is usable or not"`
	TargetTypes       []string                `db:"target_types" json:"target_types" description:"The types of entities this coupon can be applied to"`
}
