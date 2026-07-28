package events

import (
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/events"

	"github.com/disgoorg/disgo/discord"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookTeamEditData struct {
	Name       events.Changeset[string]       `json:"name" description:"The changeset of the name"`
	Short      events.Changeset[string]       `json:"short" description:"The changeset of the short description"`
	Tags       events.Changeset[[]string]     `json:"tags" description:"The changeset of the tags"`
	ExtraLinks events.Changeset[[]types.Link] `json:"extra_links" description:"The changeset of the extra links"`
	NSFW       events.Changeset[bool]         `json:"nsfw" description:"The changeset of the nsfw status"`
}

func (n WebhookTeamEditData) TargetTypes() []string {
	return []string{"team"}
}

func (n WebhookTeamEditData) Event() string {
	return "TEAM_EDIT"
}

func (n WebhookTeamEditData) Summary() string {
	return "Team Edit"
}

func (n WebhookTeamEditData) Description() string {
	return "This webhook is sent when a user edits the basic settings of a team (name/short/tags) is changed."
}

func (n WebhookTeamEditData) CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets events.Target) *discord.Embed {
	name := events.ConvertChangesetToEmbedFields[string]("Name", n.Name)
	short := events.ConvertChangesetToEmbedFields[string]("Short", n.Short)
	tags := events.ConvertChangesetToEmbedFields[[]string]("Tags", n.Tags)
	extraLinks := events.ConvertChangesetToEmbedFields[[]types.Link]("Extra Links", n.ExtraLinks)
	nsfw := events.ConvertChangesetToEmbedFields[bool]("NSFW", n.NSFW)
	return &discord.Embed{
		URL: "https://botlist.site/teams/" + targets.GetID(),
		Thumbnail: &discord.EmbedResource{
			URL: targets.GetAvatarURL(),
		},
		Title:       "📝 Team Update!",
		Description: ":heart: " + creator.DisplayName + " has updated the name/description of " + targets.GetTargetName(),
		Color:       0x8A6BFD,
		Fields: []discord.EmbedField{
			{
				Name:   "User ID:",
				Value:  creator.ID,
				Inline: validators.TruePtr,
			},
			name[0],
			name[1],
			short[0],
			short[1],
			tags[0],
			tags[1],
			extraLinks[0],
			extraLinks[1],
			nsfw[0],
			nsfw[1],
		},
	}
}

func init() {
	events.AddEvent(WebhookTeamEditData{})
}
