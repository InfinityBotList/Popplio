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

// Notification
type NotifGet struct {
	Endpoint    string           `db:"endpoint" json:"endpoint" description:"The endpoint for the subscription returned by PushSubscription"`
	NotifID     string           `db:"notif_id" json:"notif_id" description:"The ID of the notification"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at" description:"The time the notification was created"`
	UA          string           `db:"ua" json:"-"`                                                                                         // Must be parsed internally
	BrowserInfo NotifBrowserInfo `db:"-" json:"browser_info" description:"information about the browser attached to the push notification"` // Must be parsed from UA internally
}

type NotifBrowserInfo struct {
	// The OS of the browser
	OS         string `json:"os" description:"The OS of the browser"`
	Browser    string `json:"browser" description:"The browser"`
	BrowserVer string `json:"browser_ver" description:"The browser version"`
	Mobile     bool   `json:"mobile" description:"Whether the browser is on mobile or not"`
}

type NotifGetList struct {
	Notifications []NotifGet `json:"notifications"`
}
