package types

import "github.com/jackc/pgx/v5/pgtype"

type Alert struct {
	ITag      string           `db:"itag" json:"itag" validate:"required"`
	URL       pgtype.Text      `db:"url" json:"url"` // Optional
	Message   string           `db:"message" json:"message" validate:"required"`
	Type      NotificationType `db:"type" json:"type" validate:"required,oneof=success error info warning"`
	Title     string           `db:"title" json:"title" validate:"required"`
	AlertData map[string]any   `db:"alert_data" json:"alert_data"` // Optional
	Icon      string           `db:"icon" json:"icon"`             // Optional
	Priority  int              `db:"priority" json:"priority"`     // Optional
}

type AlertList struct {
	Alerts []Alert `json:"alerts" description:"List of alerts"`
}
