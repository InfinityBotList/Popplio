package events

import (
	"fmt"
	"popplio/types"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
	"github.com/infinitybotlist/eureka/jsonimpl"
	"github.com/mitchellh/mapstructure"
)

// Target is a struct to store the target of a webhook event.
//
// Similarity to rust enums: While not yet used, a webhook can technically
// have multiple targets. As such, the Target struct cannot technically be
// implemented as a simple Rust enum. In practice, as there is only one
// target per webhook, a simple enum may be possible.
//
// You can add targets here to extend the webhook system
type Target struct {
	Bot    *dovetypes.PlatformUser `json:"bot,omitempty" description:"If a bot event, the bot that the webhook is about"`
	Server *types.IndexServer      `json:"server,omitempty" description:"If a server event, the server that the webhook is about"`
	Team   *types.Team             `json:"team,omitempty" description:"If a team event, the team that the webhook is about"`
}

// The response that a webhook will recieve
type WebhookResponse struct {
	Creator  *dovetypes.PlatformUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	Type     string                  `json:"type" dynexample:"true" description:"The type of the webhook event"`
	Data     WebhookEvent            `json:"data" dynschema:"true" description:"The data of the webhook event"`
	Targets  Target                  `json:"targets" description:"The target of the webhook, can be one of. or a possible combination of bot, team and server"`
	Metadata WebhookMetadata         `json:"metadata" description:"Metadata about the webhook event"`
}

// UnmarshalJSON implements jsonimpl.Unmarshaler
//
// This is used to unmarshal the webhook response into a valid webhook event
func (wr *WebhookResponse) UnmarshalJSON(b []byte) error {
	var smap map[string]any

	err := jsonimpl.Unmarshal(b, &smap)

	if err != nil {
		return fmt.Errorf("failed to unmarshal webhook response: %w", err)
	}

	typ, ok := smap["type"].(string)

	if !ok {
		return fmt.Errorf("failed to unmarshal webhook response: type not a string")
	}

	evt, ok := eventMapToType[typ]

	if !ok {
		return fmt.Errorf("failed to unmarshal webhook response: invalid type")
	}

	// Set type and data now. Rest will be decoded into the struct
	// using mapstructure
	wr.Type = typ
	wr.Data = evt

	// decoder to copy map values to my struct using json tags
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   wr, // Save the extra data to wr
		TagName:  "json",
		Squash:   true,
	}

	decoder, e := mapstructure.NewDecoder(cfg)

	if e != nil {
		return e
	}

	// copy map to struct
	e = decoder.Decode(smap)

	if e != nil {
		return e
	}

	return nil
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

// Helper method to convert a Changeset to a set of embed fields
// for use in a discord webhook
func ConvertChangesetToEmbedFields[T any](name string, c Changeset[T]) []*discordgo.MessageEmbedField {
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
		return t.Server.ServerID
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
		if t.Server.Avatar.Path != "" {
			return t.Server.Avatar.Path
		}

		return t.Server.Avatar.DefaultPath
	}

	if t.Team != nil {
		if t.Team.Avatar.Path != "" {
			return t.Team.Avatar.Path
		}

		return t.Team.Avatar.DefaultPath
	}

	return "https://cdn.infinitybots.gg/avatars/default.webp"
}

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
