package types

import (
	"time"

	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/jackc/pgx/v5/pgtype"
)

/*
CREATE TABLE webhooks (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    url TEXT NOT NULL CHECK (url <> ''),
    secret TEXT NOT NULL CHECK (secret <> ''),
    broken BOOLEAN NOT NULL DEFAULT FALSE, -- Whether or not the webhook is broken
    simple_auth BOOLEAN NOT NULL DEFAULT FALSE, -- Whether or not the webhook should use simple auth
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (target_id, target_type)
);
*/

// @ci table=webhooks unfilled=1
//
// Represents a webhook on IBL
type Webhook struct {
	ID             pgtype.UUID `db:"id" json:"id" description:"The bot's internal ID. An artifact of database migrations."`
	Name           string      `db:"name" json:"name" description:"The name of the webhook."`
	TargetID       string      `db:"target_id" json:"target_id" description:"The target ID."`
	TargetType     string      `db:"target_type" json:"target_type" description:"The target type (bot/team etc.)."`
	Url            string      `db:"url" json:"url" description:"The URL of the webhook."`
	Broken         bool        `db:"broken" json:"broken" description:"Whether the webhook is marked as broken or not."`
	FailedRequests int         `db:"failed_requests" json:"failed_requests" description:"The number of failed requests to the webhook."`
	SimpleAuth     bool        `db:"simple_auth" json:"simple_auth" description:"Whether the webhook should use simple auth (unencrypted, just authentication headers) or not."`
	EventWhitelist []string    `db:"event_whitelist" json:"event_whitelist" description:"The events that are whitelisted for this webhook. Note that if unset, all events are whitelisted."`
	CreatedAt      time.Time   `db:"created_at" json:"created_at" description:"The time when the webhook was created."`
}

// Represents the data to be sent to create a webhook
type CreateEditWebhook struct {
	Name           string   `json:"name" description:"The name of the webhook." validate:"required"`
	Url            string   `json:"url" description:"The URL of the webhook." validate:"required"`
	Secret         string   `json:"secret" description:"The secret of the webhook, only needed for custom (non-discord) webhooks"`
	SimpleAuth     bool     `json:"simple_auth" description:"Whether the webhook should use simple auth (unencrypted, just authentication headers) or not."`
	EventWhitelist []string `json:"event_whitelist" description:"The events that are whitelisted for this webhook. Note that if unset, all events are whitelisted."`
}

type WebhookType = string

const (
	WebhookTypeText      WebhookType = "text"
	WebhookTypeTextArray WebhookType = "text[]"
	WebhookTypeLinkArray WebhookType = "link[]"
	WebhookTypeNumber    WebhookType = "number"
	WebhookTypeChangeset WebhookType = "changeset"
	WebhookTypeBoolean   WebhookType = "boolean"
)

// @ci table=webhook_logs
//
// Webhook log
type WebhookLogEntry struct {
	ID              pgtype.UUID             `db:"id" json:"id" description:"The ID of the webhook log."`
	WebhookID       pgtype.UUID             `db:"webhook_id" json:"webhook_id" description:"The ID of the webhook."`
	TargetID        string                  `db:"target_id" json:"target_id" description:"The target ID."`
	TargetType      string                  `db:"target_type" json:"target_type" description:"The target type (bot/team etc.)."`
	UserID          string                  `db:"user_id" json:"-"`
	User            *dovetypes.PlatformUser `db:"-" json:"user" description:"User ID the webhook is intended for" ci:"internal"` // Must be parsed internally
	URL             string                  `db:"url" json:"url" description:"The URL of the webhook."`
	Data            map[string]any          `db:"data" json:"data" description:"The data of the webhook."`
	Response        pgtype.Text             `db:"response" json:"response" description:"The response of the webhook request."`
	CreatedAt       time.Time               `db:"created_at" json:"created_at" description:"The time when the webhook was created."`
	State           string                  `db:"state" json:"state" description:"The state of the webhook."`
	Tries           int                     `db:"tries" json:"tries" description:"The number of send tries attempted on this webhook."`
	LastTry         time.Time               `db:"last_try" json:"last_try" description:"The time of the last send try."`
	BadIntent       bool                    `db:"bad_intent" json:"bad_intent" description:"Whether the webhook was sent with bad intent."`
	StatusCode      int                     `db:"status_code" json:"status_code" description:"The status code of the webhook request."`
	RequestHeaders  map[string]any          `db:"request_headers" json:"request_headers" description:"The headers of the webhook request."`
	ResponseHeaders map[string]any          `db:"response_headers" json:"response_headers" description:"The headers of the webhook response."`
}

type GetTestWebhookMeta struct {
	Types []TestWebhookType `json:"data" description:"The types of webhooks to test."`
}

type TestWebhookType struct {
	Type string `json:"type" description:"The type of webhook to test."`
	Data []TestWebhookVariables
}

type TestWebhookVariables struct {
	ID          string      `json:"id" description:"The ID of the variable."`
	Name        string      `json:"name" description:"The name of the variable."`
	Description string      `json:"description" description:"The description of the variable."`
	Value       string      `json:"value" description:"The default value of the variable."`
	Type        WebhookType `json:"type" description:"The type of the variable."`
}
