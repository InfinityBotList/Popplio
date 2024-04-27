package events

import (
	"popplio/state"
	"popplio/types"
	"reflect"

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
//
// All events defined under the webhooks/events folder must implement this interface
// to be considered an event
type WebhookEvent interface {
	TargetTypes() []string
	Event() string
	CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams
	Summary() string
	Description() string
}

// List of all events that have been added
var eventList = []WebhookEvent{}

// Map of event type to event
var eventMapToType = map[string]WebhookEvent{}

// Adds an event to be registered. This should be called in the init() function of the event
//
// Note that this function technically does not register the event but rather just adds it
// to the list of events to be registered.
//
// This is because `doclib` and `state` are not initialized until after state setyp
func AddEvent(a WebhookEvent) {
	eventList = append(eventList, a)
}

// Register all events that have been added
func RegisterAddedEvents() {
	for _, a := range eventList {
		state.Logger.Error("Webhook event register", zap.String("event", a.Event()))
		registerEventImpl(a)
	}
}

// Internal implementation to register an event which does the following tasks:
//
// - Adds the event to the documentation
// - Decoonstructs the event for the Test Webhooks feature
//
// Point 2 is achieved by looping over all the fields of the event
// using runtime reflection and handling changelogs/other primitive types
// where encountered. For this reason, ensure that all types used in an event
// are handled in the switch-case and add them there if not before use in an event.
//
// WARNING: This function is not concurrency safe and should only be run during initialization
func registerEventImpl(a WebhookEvent) {
	// Add the event to the map
	eventMapToType[a.Event()] = a

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

	// Helper method to generate the changeset type
	//
	// Format returned: changeset/<type>
	changesetOf := func(t types.WebhookType) types.WebhookType {
		return types.WebhookType(string(types.WebhookTypeChangeset) + "/" + string(t))
	}

	evt := EventRegistry{
		Event: a,
	}

	refType := reflect.TypeOf(a)

	var cols []types.TestWebhookVariables

	// Deconstruct the event fields to create the list of fields
	// for the test webhook feature
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
			case Changeset[bool]:
				fieldType = changesetOf(types.WebhookTypeBoolean)
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

	evt.TestVars = cols

	Registry = append(Registry, evt)
}
