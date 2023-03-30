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

type AddEvent struct {
	Event WebhookType
	Data  EventData
}

func (e *Events) AddEvents(events ...AddEvent) {
	if len(e.eventMap) == 0 {
		e.eventMap = make(map[WebhookType]EventData)
	}

	for _, data := range events {
		if data.Data.Docs == nil {
			panic("docs cannot be nil")
		}

		if data.Data.Format == nil {
			panic("format cannot be nil")
		}

		if data.Data.CreateHookParams == nil {
			panic("createhookparams cannot be nil")
		}

		docs.AddWebhook(data.Data.Docs)
		data.Data.formatTypeName = reflect.TypeOf(data.Data.Format).Name()
		e.eventMap[data.Event] = data.Data
	}
}

var RegisteredEvents = &Events{}
