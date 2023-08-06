package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeBotNewReview WebhookType = "BOT_NEW_REVIEW"

type WebhookBotNewReviewData struct {
	ReviewID string `json:"review_id" description:"The ID of the review"`
	Content  string `json:"content" description:"The content of the review"`
}

func (v WebhookBotNewReviewData) TargetType() string {
	return "bot"
}

func (n WebhookBotNewReviewData) Event() WebhookType {
	return WebhookTypeBotNewReview
}

func (n WebhookBotNewReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       "📝 New Review!",
				Description: ":heart: " + creator.DisplayName + " has left a review for " + targets.Bot.Username,
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

func (n WebhookBotNewReviewData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "NewBotReview",
		Summary: "New Bot Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user creates a new review on a bot.`,
		Format: WebhookResponse{
			Type: WebhookBotNewReviewData{}.Event(),
			Data: WebhookBotNewReviewData{},
		},
		FormatName: "WebhookResponse-WebhookNewReviewData",
	}
}

func init() {
	RegisterEvent(WebhookBotNewReviewData{})
}
