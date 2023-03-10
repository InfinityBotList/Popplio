package types

type NotificationType string

const (
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
)

type Notification struct {
	Type      NotificationType `json:"type" validate:"required,oneof=success error info warning"`
	Message   string           `json:"message" validate:"required"`
	Title     string           `json:"title" validate:"required"`
	URL       string           `json:"url"`  // Optional
	Icon      string           `json:"icon"` // Optional
	AlertData map[string]any   `json:"data"` // Optional
}
