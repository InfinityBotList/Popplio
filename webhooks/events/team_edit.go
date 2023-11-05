package events

import (
	"popplio/types"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeTeamEdit WebhookType = "TEAM_EDIT"

type WebhookTeamEditData struct {
	Name       Changeset[string]       `json:"name" description:"The changeset of the name"`
	Short      Changeset[string]       `json:"short" description:"The changeset of the short description"`
	Tags       Changeset[[]string]     `json:"tags" description:"The changeset of the tags"`
	ExtraLinks Changeset[[]types.Link] `json:"extra_links" description:"The changeset of the extra links"`
}

func (n WebhookTeamEditData) TargetType() string {
	return "team"
}

func (n WebhookTeamEditData) Event() WebhookType {
	return WebhookTypeTeamEdit
}

func (n WebhookTeamEditData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	name := convertChangesetToFields[string]("Name", n.Name)
	short := convertChangesetToFields[string]("Short", n.Short)
	tags := convertChangesetToFields[[]string]("Tags", n.Tags)
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/teams/" + targets.Team.Name,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: func() string {
						if targets.Team.Avatar.Path != "" {
							return targets.Team.Avatar.Path
						}
						return targets.Team.Avatar.DefaultPath
					}(),
				},
				Title:       "📝 Team Update!",
				Description: ":heart: " + creator.DisplayName + " has updated the name/description of your team " + targets.Team.Name,
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

func (n WebhookTeamEditData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "EditTeam",
		Summary: "Edit Team",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user edits the basic settings of a team (name/avatar) is changed.`,
		Format: WebhookResponse{
			Type: WebhookTeamEditData{}.Event(),
			Data: WebhookTeamEditData{},
		},
		FormatName: "WebhookResponse-WebhookTeamEditData",
	}
}

func init() {
	RegisterEvent(WebhookTeamEditData{})
}
