package types

import "github.com/jackc/pgx/v5/pgtype"

type AlertType string

const (
	AlertTypeSuccess AlertType = "success"
	AlertTypeError   AlertType = "error"
	AlertTypeInfo    AlertType = "info"
	AlertTypeWarning AlertType = "warning"
)

type AlertPriority int

const (
	AlertPriorityLow AlertPriority = iota
	AlertPriorityMedium
	AlertPriorityHigh
)

type Alert struct {
	ITag      pgtype.UUID    `db:"itag" json:"itag"`
	URL       pgtype.Text    `db:"url" json:"url"` // Optional
	Message   string         `db:"message" json:"message" validate:"required"`
	Type      AlertType      `db:"type" json:"type" validate:"required,oneof=success error info warning"`
	Title     string         `db:"title" json:"title" validate:"required"`
	AlertData map[string]any `db:"alert_data" json:"alert_data"`          // Optional
	Icon      string         `db:"icon" json:"icon"`                      // Optional
	Priority  AlertPriority  `db:"priority" json:"priority" enum:"1,2,3"` // Optional
}

type AlertList struct {
	Alerts []Alert `json:"alerts" description:"List of alerts"`
}
