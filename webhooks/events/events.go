package events

import (
	"fmt"
	"os"
	"popplio/types"
	"reflect"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type EventRegistry struct {
	Event    WebhookEvent
	TestVars []types.TestWebhookVariables
}

var Registry = []EventRegistry{}

// Webhook events
type WebhookEvent interface {
	TargetType() string
	Event() WebhookType
	CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams
	Docs() *docs.WebhookDoc
}

func RegisterEvent(a WebhookEvent) {
	changesetOf := func(t types.WebhookType) types.WebhookType {
		return types.WebhookType(string(types.WebhookTypeChangeset) + "/" + string(t))
	}

	evt := EventRegistry{
		Event: a,
	}

	refType := reflect.TypeOf(a)

	var cols []types.TestWebhookVariables

	for _, f := range reflect.VisibleFields(refType) {
		var fieldType string

		switch f.Type.Kind() {
		case reflect.String:
			fieldType = types.WebhookTypeText
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = types.WebhookTypeNumber
		case reflect.Struct:
			// Typeswitch here
			ti := reflect.Zero(f.Type).Interface()

			switch ti.(type) {
			case Changeset[string]:
				fieldType = changesetOf(types.WebhookTypeText)
			case Changeset[int], Changeset[int8], Changeset[int16], Changeset[int32], Changeset[int64]:
				fieldType = changesetOf(types.WebhookTypeNumber)
			default:
				panic("Illegal field type: " + string(a.Event()) + "->" + f.Name + " <struct>")
			}
		default:
			panic("Illegal field type: " + string(a.Event()) + "->" + f.Name)
		}

		if f.Tag.Get("json") == "" {
			panic("Json tag missing: " + string(a.Event()) + "->" + f.Name)
		}

		var label = f.Name

		if f.Tag.Get("testlabel") != "" {
			label = f.Tag.Get("testlabel")
		}

		cols = append(cols, types.TestWebhookVariables{
			ID:    f.Tag.Get("json"),
			Name:  label,
			Value: f.Tag.Get("testvalue"),
			Type:  fieldType,
		})
	}

	if os.Getenv("DEBUG") == "true" {
		fmt.Println(cols)
	}

	evt.TestVars = cols

	Registry = append(Registry, evt)
}

type WebhookType string

const WebhookTypeUndefined = ""

// You can add targets here to extend the webhook system
type Target struct {
	Bot    *dovetypes.PlatformUser `json:"bot,omitempty" description:"If a bot event, the bot that the webhook is about"`
	Server *types.SEO              `json:"server,omitempty" description:"If a server event, the server that the webhook is about"`
	Team   *types.Team             `json:"team,omitempty" description:"If a team event, the team that the webhook is about"`
}

// IMPL
type WebhookResponse struct {
	Creator   *dovetypes.PlatformUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	CreatedAt int64                   `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Type      WebhookType             `json:"type" dynexample:"true" description:"The type of the webhook event"`
	Data      WebhookEvent            `json:"data" dynschema:"true" description:"The data of the webhook event"`
	Targets   Target                  `json:"targets" description:"The target of the webhook, can be one of. or a possible combination of bot, team and server"`
	Metadata  WebhookMetadata         `json:"metadata" description:"Metadata about the webhook event"`
}

// Setup docs for each event
func Setup() {
	for _, event := range Registry {
		docs.AddWebhook(event.Event.Docs())
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
