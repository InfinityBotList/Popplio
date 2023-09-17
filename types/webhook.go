package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

// @ci table=webhooks, unfilled=1
//
// Webhook (omits secret)
type Webhook struct {
	ID         pgtype.UUID `db:"id" json:"id" description:"The bot's internal ID. An artifact of database migrations."`
	Url        string      `db:"url" json:"url" description:"The URL of the webhook."`
	TargetID   string      `db:"target_id" json:"target_id" description:"The target ID."`
	TargetType string      `db:"target_type" json:"target_type" description:"The target type (bot/team etc.)."`
	Broken     bool        `db:"broken" json:"broken" description:"Whether the webhook is broken."`
	CreatedAt  time.Time   `db:"created_at" json:"created_at" description:"The time when the webhook was created."`
}

type WebhookType = string

const (
	WebhookTypeText      WebhookType = "text"
	WebhookTypeNumber    WebhookType = "number"
	WebhookTypeChangeset WebhookType = "changeset"
	WebhookTypeBoolean   WebhookType = "boolean"
)

// @ci table=webhook_logs
//
// Webhook log
type WebhookLogEntry struct {
	ID         pgtype.UUID             `db:"id" json:"id" description:"The ID of the webhook log."`
	TargetID   string                  `db:"target_id" json:"target_id" description:"The target ID."`
	TargetType string                  `db:"target_type" json:"target_type" description:"The target type (bot/team etc.)."`
	UserID     string                  `db:"user_id" json:"-"`
	User       *dovetypes.PlatformUser `db:"-" json:"user" description:"User ID the webhook is intended for" ci:"internal"` // Must be parsed internally
	URL        string                  `db:"url" json:"url" description:"The URL of the webhook."`
	Data       map[string]any          `db:"data" json:"data" description:"The data of the webhook."`
	Response   pgtype.Text             `db:"response" json:"response" description:"The response of the webhook request."`
	CreatedAt  time.Time               `db:"created_at" json:"created_at" description:"The time when the webhook was created."`
	State      string                  `db:"state" json:"state" description:"The state of the webhook."`
	Tries      int                     `db:"tries" json:"tries" description:"The number of send tries attempted on this webhook."`
	LastTry    time.Time               `db:"last_try" json:"last_try" description:"The time of the last send try."`
	BadIntent  bool                    `db:"bad_intent" json:"bad_intent" description:"Whether the webhook was sent with bad intent."`
	StatusCode int                     `db:"status_code" json:"status_code" description:"The status code of the webhook request."`
}

type PatchWebhook struct {
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`
	Clear         bool   `json:"clear"`
}

type GetTestWebhookMeta struct {
	Types []TestWebhookType `json:"data"`
}

type TestWebhookType struct {
	Type string `json:"type" description:"The type of webhook to test."`
	Data []TestWebhookVariables
}

type TestWebhookVariables struct {
	ID    string      `json:"id" description:"The ID of the variable."`
	Name  string      `json:"name" description:"The name of the variable."`
	Value string      `json:"value" description:"The default value of the variable."`
	Type  WebhookType `json:"type" description:"The type of the variable."`
}
