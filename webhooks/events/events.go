package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
)

// Webhook events
type WebhookEvent interface {
	Event() WebhookType
	CreateHookParams(creator *dovewing.DiscordUser, targets Target) *discordgo.WebhookParams
}

type WebhookType string

type Target struct {
	Bot *dovewing.DiscordUser `json:"bot,omitempty" description:"If a bot event, the bot that the webhook is about"`
}

// IMPL
type WebhookResponse[E WebhookEvent] struct {
	Creator   *dovewing.DiscordUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	CreatedAt int64                 `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Type      WebhookType           `json:"type" dynexample:"true"`
	Data      E                     `json:"data" dynschema:"true"`
	Targets   Target                `json:"targets" description:"The target of the webhook, can be one of. or a possible combination of bot, team and server"`
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
