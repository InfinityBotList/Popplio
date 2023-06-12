package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
)

const webhookTypeBotNewReview WebhookType = "BOT_NEW_REVIEW"

type WebhookBotNewReviewData struct {
	ReviewID string `json:"review_id"` // The ID of the review
	Content  string `json:"content"`   // The content of the review
}

func (n WebhookBotNewReviewData) Event() WebhookType {
	return webhookTypeBotNewReview
}

func (n WebhookBotNewReviewData) CreateHookParams(creator *dovewing.DiscordUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       "📝 New Review!",
				Description: ":heart:" + creator.Username + "#" + creator.Discriminator + " has left a review for " + targets.Bot.Username,
				Color:       0x8A6BFD,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Review ID:",
						Value:  n.ReviewID,
						Inline: true,
					},
					{
						Name:   "User ID:",
						Value:  creator.ID,
						Inline: true,
					},
					{
						Name: "Review Content",
						Value: func() string {
							if len(n.Content) > 1000 {
								return n.Content[:1000] + "..."
							}

							return n.Content
						}(),
						Inline: true,
					},
					{
						Name:   "Review Page",
						Value:  "[View " + targets.Bot.Username + "](https://botlist.site/" + targets.Bot.ID + ")",
						Inline: true,
					},
				},
			},
		},
	}
}

func init() {
	AddEvent(&docs.WebhookDoc{
		Name:    "NewBotReview",
		Summary: "New Bot Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user creates a new review on a bot.

The data of the webhook may differ based on its webhook type

If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
`,
		Format: WebhookResponse[WebhookBotNewReviewData]{
			Type: WebhookBotNewReviewData{}.Event(),
			Data: WebhookBotNewReviewData{},
		},
		FormatName: "WebhookResponse-WebhookNewReviewData",
	})
}
