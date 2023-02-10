package partners

import "popplio/types"

type Partner struct {
	ID     string       `json:"id" validate:"required"`
	Name   string       `json:"name" validate:"required"`
	Image  string       `json:"image" validate:"required"`
	Short  string       `json:"short" validate:"required"`
	Links  []types.Link `json:"links" validate:"required,min=1,max=2"`
	UserID string       `json:"-" validate:"required,numeric"`

	// Internal field
	User *types.DiscordUser `json:"user"`
}

type PartnerList struct {
	Featured        []*Partner `json:"featured" validate:"required,dive"`
	BotPartners     []*Partner `json:"bot_partners" validate:"required,dive"`
	BotListPartners []*Partner `json:"bot_list_partners" validate:"required,dive"`
}
