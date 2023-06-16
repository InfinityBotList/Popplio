package events

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
)

const webhookTypeBotVote WebhookType = "BOT_VOTE"

type WebhookBotVoteData struct {
	Votes int  `json:"votes"` // The amount of votes the bot received
	Test  bool `json:"test"`  // Whether the vote was a test vote or not
}

func (v WebhookBotVoteData) Event() WebhookType {
	return webhookTypeBotVote
}

func (v WebhookBotVoteData) CreateHookParams(creator *dovewing.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart: " + creator.DisplayName + " has voted for " + targets.Bot.Username,
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
						Name:   "Vote Page",
						Value:  "[View " + targets.Bot.Username + "](https://botlist.site/" + targets.Bot.ID + ")",
						Inline: true,
					},
					{
						Name:   "Vote Page",
						Value:  "[Vote for " + targets.Bot.Username + "](https://botlist.site/" + targets.Bot.ID + "/vote)",
						Inline: true,
					},
				},
			},
		},
	}
}

func init() {
	AddEvent(&docs.WebhookDoc{
		Name:    "NewBotVote",
		Summary: "New Bot Vote",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user votes for a bot.`,
		Format: WebhookResponse[WebhookBotVoteData]{
			Type: WebhookBotVoteData{}.Event(),
			Data: WebhookBotVoteData{},
		},
		FormatName: "WebhookResponse-WebhookBotVoteData",
	})
}
