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
	Avatar     Changeset[string]       `json:"avatar" description:"The changeset of the avatar"`
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
