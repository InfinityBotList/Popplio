package events

import (
	"fmt"
	"os"
	"popplio/state"
	"popplio/types"
	"reflect"
	"time"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"go.uber.org/zap"
)

type EventRegistry struct {
	Event    WebhookEvent
	TestVars []types.TestWebhookVariables
}

var Registry = []EventRegistry{}

// Webhook events
type WebhookEvent interface {
	TargetTypes() []string
	Event() string
	CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams
	Summary() string
	Description() string
}

var eventList = []WebhookEvent{}

// Adds an event to be registered. This should be called in the init() function of the event
//
// Note that this function does not register the event, it just adds it to the list of events to be registered.
// This is because `doclib` and `state` are not initialized until after state setyp
func RegisterEvent(a WebhookEvent) {
	eventList = append(eventList, a)
}

// Register all events
func RegisterAllEvents() {
	for _, a := range eventList {
		state.Logger.Error("Webhook event register", zap.String("event", a.Event()))
		registerEventImpl(a)
	}
}

// Internal implementation to register an event
func registerEventImpl(a WebhookEvent) {
	docs.AddWebhook(&docs.WebhookDoc{
		Name:    a.Event(),
		Summary: a.Summary(),
		Tags: []string{
			"Webhooks",
		},
		Description: a.Description(),
		Format: WebhookResponse{
			Type: a.Event(),
			Data: a,
		},
		FormatName: "WEBHOOK-" + a.Event(),
	})

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
		case reflect.Bool:
			fieldType = types.WebhookTypeBoolean
		case reflect.Struct:
			// Typeswitch here
			ti := reflect.Zero(f.Type).Interface()

			switch ti.(type) {
			case Changeset[[]string]:
				fieldType = changesetOf(types.WebhookTypeTextArray)
			case Changeset[[]types.Link]:
				fieldType = changesetOf(types.WebhookTypeLinkArray)
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
			ID:          f.Tag.Get("json"),
			Name:        label,
			Description: f.Tag.Get("description"),
			Value:       f.Tag.Get("testvalue"),
			Type:        fieldType,
		})
	}

	if os.Getenv("DEBUG") == "true" {
		fmt.Println(cols)
	}

	evt.TestVars = cols

	Registry = append(Registry, evt)
}

// You can add targets here to extend the webhook system
type Target struct {
	Bot    *dovetypes.PlatformUser `json:"bot,omitempty" description:"If a bot event, the bot that the webhook is about"`
	Server *types.SEO              `json:"server,omitempty" description:"If a server event, the server that the webhook is about"`
	Team   *types.Team             `json:"team,omitempty" description:"If a team event, the team that the webhook is about"`
}

type WebhookResponse struct {
	Creator  *dovetypes.PlatformUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	Type     string                  `json:"type" dynexample:"true" description:"The type of the webhook event"`
	Data     WebhookEvent            `json:"data" dynschema:"true" description:"The data of the webhook event"`
	Targets  Target                  `json:"targets" description:"The target of the webhook, can be one of. or a possible combination of bot, team and server"`
	Metadata WebhookMetadata         `json:"metadata" description:"Metadata about the webhook event"`
}

// Core structs
// A changeset represents a change in a value
type Changeset[T any] struct {
	Old T `json:"old"`
	New T `json:"new"`
}

type WebhookMetadata struct {
	CreatedAt int64 `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Test      bool  `json:"test" description:"Whether the vote was a test vote or not"`
}

// Given a webhook metadata object, parse it and return a valid/parsed one
//
// The created_at field will be set to the current time IF it is not set
func ParseWebhookMetadata(w *WebhookMetadata) WebhookMetadata {
	if w == nil {
		w = &WebhookMetadata{}
	}

	if w.CreatedAt == 0 {
		w.CreatedAt = time.Now().Unix()
	}

	return *w
}

func convertChangesetToFields[T any](name string, c Changeset[T]) []*discordgo.MessageEmbedField {
	return []*discordgo.MessageEmbedField{
		{
			Name: "Old " + name,
			Value: func() string {
				if len(fmt.Sprint(c.Old)) > 1000 {
					return fmt.Sprint(c.Old)[:1000] + "..."
				}

				return fmt.Sprint(c.Old)
			}(),
			Inline: true,
		},
		{
			Name: "New " + name,
			Value: func() string {
				if len(fmt.Sprint(c.New)) > 1000 {
					return fmt.Sprint(c.New)[:1000] + "..."
				}

				return fmt.Sprint(c.New)
			}(),
			Inline: true,
		},
	}
}

// Abstract fetching to make events easier to implement

// Gets the best/single target type of a webhook event
func (t Target) GetBestTargetType() string {
	if t.Bot != nil {
		return "bot"
	}

	if t.Server != nil {
		return "server"
	}

	if t.Team != nil {
		return "team"
	}

	return "<unknown>"
}

// Get the target types of a webhook event
func (t Target) GetTargetTypes() []string {
	var types []string

	if t.Bot != nil {
		types = append(types, "bot")
	}

	if t.Server != nil {
		types = append(types, "server")
	}

	if t.Team != nil {
		types = append(types, "team")
	}

	return types
}

// Gets the ID of a target
func (t Target) GetID() string {
	if t.Bot != nil {
		return t.Bot.ID
	}

	if t.Server != nil {
		return t.Server.ID
	}

	if t.Team != nil {
		return t.Team.ID
	}

	return "<unknown>"
}

// Get the username of a target
func (t Target) GetUsername() string {
	if t.Bot != nil {
		return t.Bot.Username
	}

	if t.Server != nil {
		return t.Server.Name
	}

	if t.Team != nil {
		return t.Team.Name
	}

	return "<unknown>"
}

// Get the display name of a target
func (t Target) GetDisplayName() string {
	if t.Bot != nil {
		return t.Bot.DisplayName
	}

	if t.Server != nil {
		return t.Server.Name
	}

	if t.Team != nil {
		return t.Team.Name
	}

	return "<unknown>"
}

// Get the avatar URL of a target
func (t Target) GetAvatarURL() string {
	if t.Bot != nil {
		return t.Bot.Avatar
	}

	if t.Server != nil {
		return t.Server.Avatar
	}

	if t.Team != nil {
		if t.Team.Avatar.Path != "" {
			return t.Team.Avatar.Path
		}

		return t.Team.Avatar.DefaultPath
	}

	return "https://cdn.infinitybots.gg/avatars/default.webp"
}

// Helper abstractions on target

// Returns the target name'. Currently <target type> <username>
func (t Target) GetTargetName() string {
	return t.GetBestTargetType() + " " + t.GetUsername()
}

// Returns a link to the target
func (t Target) GetTargetLink(header, path string) string {
	// Teams do not support vanities at this time
	if t.Team != nil {
		return "[" + header + " " + t.GetUsername() + "](https://botlist.site/teams/" + t.GetID() + path + ")"
	}

	return "[" + header + " " + t.GetUsername() + "](https://botlist.site/" + t.GetID() + path + ")"
}

// Shorthand for t.GetTargetLink("View", "")
func (t Target) GetViewLink() string {
	return t.GetTargetLink("View", "")
}
