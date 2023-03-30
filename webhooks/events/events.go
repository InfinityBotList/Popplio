// A set of common event handling for webhook responses
package events

import (
	"errors"
	"reflect"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/doclib"
)

func (w *WebhookResponse) Validate() (*EventData, error) {
	evd, ok := RegisteredEvents.eventMap[w.Type]

	if !ok {
		return nil, errors.New("invalid webhook type")
	}

	// Cast to evd.Format
	if reflect.TypeOf(w.Data).Name() != evd.formatTypeName {
		return nil, errors.New("invalid webhook data")
	}

	return &evd, nil
}

type EventData struct {
	Docs             *docs.WebhookDoc
	Format           any
	CreateHookParams func(w *WebhookResponse) *discordgo.WebhookParams
	formatTypeName   string
}

type Events struct {
	eventMap map[WebhookType]EventData
}

func (e *Events) AddEvent(event WebhookType, data EventData) {
	if len(e.eventMap) == 0 {
		e.eventMap = make(map[WebhookType]EventData)
	}

	if data.Docs == nil {
		panic("docs cannot be nil")
	}

	if data.Format == nil {
		panic("format cannot be nil")
	}

	if data.CreateHookParams == nil {
		panic("createhookparams cannot be nil")
	}

	docs.AddWebhook(data.Docs)
	data.formatTypeName = reflect.TypeOf(data.Format).Name()
	e.eventMap[event] = data
}

var RegisteredEvents = &Events{}
