package events

import (
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookTeamEditData struct {
	Name       Changeset[string]       `json:"name" description:"The changeset of the name"`
	Short      Changeset[string]       `json:"short" description:"The changeset of the short description"`
	Tags       Changeset[[]string]     `json:"tags" description:"The changeset of the tags"`
	ExtraLinks Changeset[[]types.Link] `json:"extra_links" description:"The changeset of the extra links"`
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

func (n WebhookTeamEditData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	name := convertChangesetToFields[string]("Name", n.Name)
	short := convertChangesetToFields[string]("Short", n.Short)
	tags := convertChangesetToFields[[]string]("Tags", n.Tags)
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
	RegisterEvent(WebhookTeamEditData{})
}
