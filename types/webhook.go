package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

// @ci table=webhook_logs
//
// Webhook log
type WebhookLogEntry struct {
	ID          pgtype.UUID             `db:"id" json:"id" description:"The ID of the webhook log."`
	TargetID    string                  `db:"target_id" json:"target_id" description:"The target ID."`
	TargetType  string                  `db:"target_type" json:"target_type" description:"The target type (bot/team etc.)."`
	UserID      string                  `db:"user_id" json:"-"`
	User        *dovetypes.PlatformUser `json:"user" description:"User ID the webhook is intended for" ci:"internal"` // Must be parsed internally
	URL         string                  `db:"url" json:"url" description:"The URL of the webhook."`
	Data        map[string]any          `db:"data" json:"data" description:"The data of the webhook."`
	Sign        string                  `db:"sign" json:"sign" description:"The auth secret of the webhook."`
	CreatedAt   time.Time               `db:"created_at" json:"created_at" description:"The time when the webhook was created."`
	State       string                  `db:"state" json:"state" description:"The state of the webhook."`
	Tries       int                     `db:"tries" json:"tries" description:"The number of send tries attempted on this webhook."`
	LastTry     time.Time               `db:"last_try" json:"last_try" description:"The time of the last send try."`
	BadIntent   bool                    `db:"bad_intent" json:"bad_intent" description:"Whether the webhook was sent with bad intent."`
	UseInsecure bool                    `db:"use_insecure" json:"use_insecure" description:"Whether the webhook should be sent with an insecure connection."`
}

type PatchBotWebhook struct {
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`
	WebhooksV2    *bool  `json:"webhooks_v2"`
	Clear         bool   `json:"clear"`
}

type PatchTeamWebhook struct {
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`
	Clear         bool   `json:"clear"`
}
