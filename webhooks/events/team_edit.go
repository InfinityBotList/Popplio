package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const webhookTypeTeamEdit WebhookType = "TEAM_EDIT"

type WebhookTeamEditData struct {
	Name   Changeset[string] `json:"name"`   // The changeset of the name
	Avatar Changeset[string] `json:"avatar"` // The changeset of the avatar
}

func (n WebhookTeamEditData) Event() WebhookType {
	return webhookTypeTeamEdit
}

func (n WebhookTeamEditData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/teams/" + targets.Team.Name,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Team.Avatar,
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
					{
						Name: "Old Name",
						Value: func() string {
							if len(n.Name.Old) > 1000 {
								return n.Name.Old[:1000] + "..."
							}

							return n.Name.Old
						}(),
						Inline: true,
					},
					{
						Name: "New Name",
						Value: func() string {
							if len(n.Name.New) > 1000 {
								return n.Name.New[:1000] + "..."
							}

							return n.Name.New
						}(),
						Inline: true,
					},
					{
						Name: "Old Avatar",
						Value: func() string {
							if len(n.Avatar.Old) > 1000 {
								return n.Avatar.Old[:1000] + "..."
							}

							return n.Avatar.Old
						}(),
					},
					{
						Name: "New Avatar",
						Value: func() string {
							if len(n.Avatar.New) > 1000 {
								return n.Avatar.New[:1000] + "..."
							}

							return n.Avatar.New
						}(),
					},
				},
			},
		},
	}
}

func init() {
	AddEvent(&docs.WebhookDoc{
		Name:    "EditTeam",
		Summary: "Edit Team",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user edits the basic settings of a team (name/avatar) is changed.`,
		Format: WebhookResponse[WebhookTeamEditData]{
			Type: WebhookTeamEditData{}.Event(),
			Data: WebhookTeamEditData{},
		},
		FormatName: "WebhookResponse-WebhookTeamEditData",
	})
}
