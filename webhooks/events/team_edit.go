package events

import (
	"popplio/types"
	"popplio/webhooks/core/events"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookTeamEditData struct {
	Name       events.Changeset[string]       `json:"name" description:"The changeset of the name"`
	Short      events.Changeset[string]       `json:"short" description:"The changeset of the short description"`
	Tags       events.Changeset[[]string]     `json:"tags" description:"The changeset of the tags"`
	ExtraLinks events.Changeset[[]types.Link] `json:"extra_links" description:"The changeset of the extra links"`
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

func (n WebhookTeamEditData) CreateHookParams(creator *dovetypes.PlatformUser, targets events.Target) *discordgo.WebhookParams {
	name := events.ConvertChangesetToFields[string]("Name", n.Name)
	short := events.ConvertChangesetToFields[string]("Short", n.Short)
	tags := events.ConvertChangesetToFields[[]string]("Tags", n.Tags)
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/teams/" + targets.GetID(),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.GetAvatarURL(),
				},
				Title:       "📝 Team Update!",
				Description: ":heart: " + creator.DisplayName + " has updated the name/description of " + targets.GetTargetName(),
				Color:       0x8A6BFD,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User ID:",
						Value:  creator.ID,
						Inline: true,
					},
					name[0],
					name[1],
					short[0],
					short[1],
					tags[0],
					tags[1],
				},
			},
		},
	}
}

func init() {
	events.RegisterEvent(WebhookTeamEditData{})
}
