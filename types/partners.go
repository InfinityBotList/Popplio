package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// @ci table=partners
//
// Partner represents a IBL partner.
type Partner struct {
	ID        string                  `db:"id" json:"id" description:"The partners ID" validate:"required"`
	Name      string                  `db:"name" json:"name" description:"The partners name" validate:"required"`
	Avatar    *AssetMetadata          `db:"-" json:"avatar" description:"The partners avatar" ci:"internal"` // Must be parsed internally
	Short     string                  `db:"short" json:"short" description:"Short description of the partner" validate:"required"`
	Links     []Link                  `db:"links" json:"links" description:"Links of the partners" validate:"required,min=1,max=2"`
	Type      string                  `db:"type" json:"type" description:"Type of partner" validate:"required"`
	CreatedAt time.Time               `db:"created_at" json:"created_at" description:"When the partner was created on DB" validate:"required"`
	UserID    string                  `db:"user_id" json:"-" description:"User ID of the partner. Is an internal field" validate:"required"`
	User      *dovetypes.PlatformUser `db:"-" json:"user" description:"The partner's user information" ci:"internal"` // Must be parsed internally
	BotID     *string                 `db:"bot_id" json:"bot_id" description:"The bot ID that is associated with the partner"`
}

// @ci table=partner_types
//
// PartnerTypes represents a IBL partner type.
type PartnerTypes struct {
	ID        string    `db:"id" json:"id" description:"The partner type ID"`
	Name      string    `db:"name" json:"name" description:"The partner type name"`
	Short     string    `db:"short" json:"short" description:"Short description of the partner type"`
	Icon      string    `db:"icon" json:"icon" description:"Iconify icon of the partner type"`
	CreatedAt time.Time `db:"created_at" json:"created_at" description:"When the partner type was created on DB"`
}

type PartnerList struct {
	Partners     []Partner      `json:"partners"`
	PartnerTypes []PartnerTypes `json:"partner_types"`
}
