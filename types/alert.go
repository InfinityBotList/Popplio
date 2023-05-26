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
	ITag      pgtype.UUID        `db:"itag" json:"itag" description:"The alerts ID, while this was originally a db migration artifact, it is now the de-facto ID."`
	URL       pgtype.Text        `db:"url" json:"url" description:"The URL to send the alert to"` // Optional
	Message   string             `db:"message" json:"message" validate:"required"`
	Type      AlertType          `db:"type" json:"type" validate:"required,oneof=success error info warning"`
	Title     string             `db:"title" json:"title" validate:"required"`
	CreatedAt pgtype.Timestamptz `db:"created_at" json:"created_at" description:"The alert's creation date"`
	Acked     bool               `db:"acked" json:"acked" description:"Whether the alert has been acknowledged"`
	AlertData map[string]any     `db:"alert_data" json:"alert_data"`          // Optional
	Icon      string             `db:"icon" json:"icon"`                      // Optional
	Priority  AlertPriority      `db:"priority" json:"priority" enum:"1,2,3"` // Optional
}

type AlertList struct {
	Alerts []Alert `json:"alerts" description:"List of alerts"`
}
