package events

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeBotVote WebhookType = "BOT_VOTE"

type WebhookBotVoteData struct {
	Votes   int `json:"votes" description:"The number of votes the bot received"`
	PerUser int `json:"per_user" description:"The number of votes the user has given"`
}

func (v WebhookBotVoteData) TargetType() string {
	return "bot"
}

func (v WebhookBotVoteData) Event() WebhookType {
	return WebhookTypeBotVote
}

func (v WebhookBotVoteData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       "🎉 Vote Count Updated!",
				Description: ":heart: " + creator.DisplayName + " has voted for *bot*: " + targets.Bot.Username,
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

func (v WebhookBotVoteData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "NewBotVote",
		Summary: "New Bot Vote",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user votes for a bot.`,
		Format: WebhookResponse{
			Type: WebhookBotVoteData{}.Event(),
			Data: WebhookBotVoteData{},
		},
		FormatName: "WebhookResponse-WebhookBotVoteData",
	}
}

func init() {
	RegisterEvent(WebhookBotVoteData{})
}
