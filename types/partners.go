package types

import "github.com/infinitybotlist/eureka/dovewing/dovetypes"

// @ci table=partners
//
// Partner represents a IBL partner.
type Partner struct {
	ID     string                  `db:"id" json:"id" description:"The partners ID" validate:"required"`
	Name   string                  `db:"name" json:"name" description:"The partners name" validate:"required"`
	Image  string                  `db:"image" json:"image" description:"URL of the partners image" validate:"required"`
	Short  string                  `db:"short" json:"short" description:"Short description of the partner" validate:"required"`
	Links  []Link                  `db:"links" json:"links" description:"Links of the partners" validate:"required,min=1,max=2"`
	Type   string                  `db:"type" json:"type" description:"Type of partner" validate:"required"`
	UserID string                  `db:"user_id" json:"-" description:"User ID of the partner. Is an internal field" validate:"required"`
	User   *dovetypes.PlatformUser `db:"-" json:"user" description:"The partner's user information" ci:"internal"` // Must be parsed internally
}

type PartnerList struct {
	Partners []Partner `json:"partners"`
}
