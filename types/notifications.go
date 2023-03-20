package types

import "time"

type NotificationType string

const (
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
)

type NotificationInfo struct {
	PublicKey string `json:"public_key"`
}

// A user subscription for push notifications
type UserSubscription struct {
	Auth     string `json:"auth" description:"The auth key for the subscription returned by PushSubscription"`
	P256dh   string `json:"p256dh" description:"The p256dh key for the subscription returned by PushSubscription"`
	Endpoint string `json:"endpoint" description:"The endpoint for the subscription returned by PushSubscription"`
}

type Notification struct {
	Type      NotificationType `json:"type" validate:"required,oneof=success error info warning"`
	Message   string           `json:"message" validate:"required"`
	Title     string           `json:"title" validate:"required"`
	Priority  int              `json:"priority"` // Optional
	URL       string           `json:"url"`      // Optional
	Icon      string           `json:"icon"`     // Optional
	AlertData map[string]any   `json:"data"`     // Optional
}

// Notification
type NotifGet struct {
	Endpoint    string           `json:"endpoint"`
	NotifID     string           `json:"notif_id"`
	CreatedAt   time.Time        `json:"created_at"`
	BrowserInfo NotifBrowserInfo `json:"browser_info"`
}

type NotifBrowserInfo struct {
	// The OS of the browser
	OS         string
	Browser    string
	BrowserVer string
	Mobile     bool
}

type NotifGetList struct {
	Notifications []NotifGet `json:"notifications"`
}
