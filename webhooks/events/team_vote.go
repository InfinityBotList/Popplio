package events

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeTeamVote WebhookType = "TEAM_VOTE"

type WebhookTeamVoteData struct {
	Votes    int  `json:"votes" description:"The number of votes the team received"`
	Downvote bool `json:"downvote" description:"Whether the vote was a downvote"`
	PerUser  int  `json:"per_user" description:"The number of votes the user has given"`
}

func (n WebhookTeamVoteData) TargetType() string {
	return "team"
}

func (v WebhookTeamVoteData) Event() WebhookType {
	return WebhookTypeTeamVote
}

func (v WebhookTeamVoteData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Team.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: func() string {
						if targets.Team.Avatar.Path != "" {
							return targets.Team.Avatar.Path
						}
						return targets.Team.Avatar.DefaultPath
					}(),
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart: " + creator.DisplayName + " has voted for *team*: " + targets.Team.Name,
				Color:       0x8A6BFD,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Vote Count:",
						Value:  strconv.Itoa(int(v.Votes)),
						Inline: true,
					},
					{
						Name:   "User ID:",
						Value:  creator.ID,
						Inline: true,
					},
					{
						Name:   "View Page",
						Value:  "[View " + targets.Team.Name + "](https://botlist.site/teams/" + targets.Team.ID + ")",
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[Vote for " + targets.Team.Name + "](https://botlist.site/teams/" + targets.Team.ID + "/vote)",
						Inline: true,
					},
				},
			},
		},
	}
}

func (v WebhookTeamVoteData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "NewTeamVote",
		Summary: "New Team Vote",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user votes for a team.`,
		Format: WebhookResponse{
			Type: WebhookTeamVoteData{}.Event(),
			Data: WebhookTeamVoteData{},
		},
		FormatName: "WebhookResponse-WebhookTeamVoteData",
	}
}

func init() {
	RegisterEvent(WebhookTeamVoteData{})
}
