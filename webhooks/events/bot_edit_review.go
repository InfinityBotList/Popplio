package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
)

const webhookTypeBotEditReview WebhookType = "BOT_EDIT_REVIEW"

type WebhookBotEditReviewData struct {
	ReviewID   string `json:"review_id"`   // The ID of the review
	OldContent string `json:"old_content"` // The old content of the review
	NewContent string `json:"new_content"` // The new content of the review
}

func (n WebhookBotEditReviewData) Event() WebhookType {
	return webhookTypeBotEditReview
}

func (n WebhookBotEditReviewData) CreateHookParams(creator *dovewing.DiscordUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       "📝 Review Editted!",
				Description: ":heart:" + creator.Username + "#" + creator.Discriminator + " has editted a review for " + targets.Bot.Username,
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
						Name: "Old Content",
						Value: func() string {
							if len(n.OldContent) > 1000 {
								return n.OldContent[:1000] + "..."
							}

							return n.OldContent
						}(),
						Inline: true,
					},
					{
						Name: "New Content",
						Value: func() string {
							if len(n.NewContent) > 1000 {
								return n.NewContent[:1000] + "..."
							}

							return n.NewContent
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
		Name:    "EditBotReview",
		Summary: "Edit Bot Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user edits an existing review on a bot.

The data of the webhook may differ based on its webhook type

If the webhook type is WebhookTypeEditReview, the data will be of type WebhookEditReviewData
`,
		Format: WebhookResponse[WebhookBotEditReviewData]{
			Type: WebhookBotEditReviewData{}.Event(),
			Data: WebhookBotEditReviewData{},
		},
		FormatName: "WebhookResponse-WebhookEditReviewData",
	})
}
