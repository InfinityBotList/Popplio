package events

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeServerVote WebhookType = "SERVER_VOTE"

type WebhookServerVoteData struct {
	Votes    int  `json:"votes" description:"The number of votes the server received"`
	Downvote bool `json:"downvote" description:"Whether the vote was a downvote"`
	PerUser  int  `json:"per_user" description:"The number of votes the user has given"`
}

func (n WebhookServerVoteData) TargetType() string {
	return "server"
}

func (v WebhookServerVoteData) Event() WebhookType {
	return WebhookTypeServerVote
}

func (v WebhookServerVoteData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Server.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Server.Avatar,
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart: " + creator.DisplayName + " has voted for *server*: " + targets.Server.Name,
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
						Value:  "[View " + targets.Server.Name + "](https://botlist.site/server/" + targets.Server.ID + ")",
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[Vote for " + targets.Server.Name + "](https://botlist.site/server/" + targets.Server.ID + "/vote)",
						Inline: true,
					},
				},
			},
		},
	}
}

func (v WebhookServerVoteData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "NewServerVote",
		Summary: "New Server Vote",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user votes for a server.`,
		Format: WebhookResponse{
			Type: WebhookServerVoteData{}.Event(),
			Data: WebhookServerVoteData{},
		},
		FormatName: "WebhookResponse-WebhookServerVoteData",
	}
}

func init() {
	RegisterEvent(WebhookServerVoteData{})
}
