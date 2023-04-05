package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// The state of the webhook in the database
type WebhookSaveState int

const (
	WebhookSaveStatePending WebhookSaveState = iota
	WebhookSaveStateSuccess
	WebhookSaveStateFailed
	WebhookSaveStateRemoved
)

type WebhookEntityType int

const (
	WebhookEntityTypeBot WebhookEntityType = iota
)

// Webhook log
type WebhookLog struct {
	ID         pgtype.UUID       `db:"id" json:"id" description:"The ID of the webhook log."`
	EntityID   pgtype.UUID       `db:"entity_id" json:"entity_id" description:"The entities ID."`
	EntityType WebhookEntityType `db:"entity_type" json:"entity_type" description:"The type of the entity."`
	UserID     pgtype.UUID       `db:"user_id" json:"user_id" description:"The user ID triggering the hook"`
	URL        string            `db:"url" json:"url" description:"The URL of the webhook."`
	Data       string            `db:"data" json:"data" description:"The data of the webhook."`
	Sign       string            `db:"sign" json:"sign" description:"The auth secret of the webhook."`
	CreatedAt  time.Time         `db:"created_at" json:"created_at" description:"The time when the webhook was created."`
	State      WebhookSaveState  `db:"state" json:"state" description:"The state of the webhook."`
	Tries      int               `db:"tries" json:"tries" description:"The number of send tries attempted on this webhook."`
	LastTry    time.Time         `db:"last_try" json:"last_try" description:"The time of the last send try."`
}
