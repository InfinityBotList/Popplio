package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeBotEditReview WebhookType = "BOT_EDIT_REVIEW"

type WebhookBotEditReviewData struct {
	ReviewID string            `json:"review_id" description:"The ID of the review"`
	Content  Changeset[string] `json:"content" description:"The content of the review"`
}

func (n WebhookBotEditReviewData) Event() WebhookType {
	return WebhookTypeBotEditReview
}

func (n WebhookBotEditReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       "📝 Review Editted!",
				Description: ":heart: " + creator.DisplayName + " has editted a review for " + targets.Bot.Username,
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
							if len(n.Content.Old) > 1000 {
								return n.Content.Old[:1000] + "..."
							}

							return n.Content.Old
						}(),
						Inline: true,
					},
					{
						Name: "New Content",
						Value: func() string {
							if len(n.Content.New) > 1000 {
								return n.Content.New[:1000] + "..."
							}

							return n.Content.New
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
		Description: `This webhook is sent when a user edits an existing review on a bot.`,
		Format: WebhookResponse[WebhookBotEditReviewData]{
			Type: WebhookBotEditReviewData{}.Event(),
			Data: WebhookBotEditReviewData{},
		},
		FormatName: "WebhookResponse-WebhookEditReviewData",
	})
}
