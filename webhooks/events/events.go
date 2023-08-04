package events

import (
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

// Webhook events
type WebhookEvent interface {
	Event() WebhookType
	CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams
}

type WebhookType string

// You can add targets here to extend the webhook system
type Target struct {
	Bot  *dovetypes.PlatformUser `json:"bot,omitempty" description:"If a bot event, the bot that the webhook is about"`
	Team *types.PartialTeam      `json:"team,omitempty" description:"If a team event, the team that the webhook is about"`
}

// IMPL
type WebhookResponse[E WebhookEvent] struct {
	Creator   *dovetypes.PlatformUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	CreatedAt int64                   `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Type      WebhookType             `json:"type" dynexample:"true" description:"The type of the webhook event"`
	Data      E                       `json:"data" dynschema:"true" description:"The data of the webhook event"`
	Targets   Target                  `json:"targets" description:"The target of the webhook, can be one of. or a possible combination of bot, team and server"`
	Metadata  WebhookMetadata         `json:"metadata" description:"Metadata about the webhook event"`
}

// Setup docs for each event
var eventDocs = []func(){}

func AddEvent(wdoc *docs.WebhookDoc) {
	eventDocs = append(eventDocs, func() {
		docs.AddWebhook(wdoc)
	})
}

func Setup() {
	for _, event := range eventDocs {
		event()
	}
}

// Core structs
// A changeset represents a change in a value
type Changeset[T any] struct {
	Old T `json:"old"`
	New T `json:"new"`
}

type WebhookMetadata struct {
	Test bool `json:"test" description:"Whether the vote was a test vote or not"`
}

func DefaultWebhookMetadata() *WebhookMetadata {
	return &WebhookMetadata{
		Test: false,
	}
}

func ParseWebhookMetadata(w *WebhookMetadata) WebhookMetadata {
	if w == nil {
		w = DefaultWebhookMetadata()
	}

	return *w
}
