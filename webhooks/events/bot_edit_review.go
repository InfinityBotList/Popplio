package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeBotEditReview WebhookType = "BOT_EDIT_REVIEW"

type WebhookBotEditReviewData struct {
	ReviewID string            `json:"review_id" description:"The ID of the review"`
	Stars    Changeset[int32]  `json:"stars" description:"The number of stars the auther gave to the review"`
	Content  Changeset[string] `json:"content" description:"The content of the review"`
}

func (v WebhookBotEditReviewData) TargetType() string {
	return "bot"
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
				Description: ":heart: " + creator.DisplayName + " has editted a review for bot " + targets.Bot.Username,
				Color:       0x8A6BFD,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Review ID",
						Value:  n.ReviewID,
						Inline: true,
					},
					{
						Name:   "User ID",
						Value:  creator.ID,
						Inline: true,
					},
					{
						Name:  "Stars",
						Value: fmt.Sprintf("%d/5 -> %d/5", n.Stars.Old, n.Stars.New),
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

func (n WebhookBotEditReviewData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "EditBotReview",
		Summary: "Edit Bot Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user edits an existing review on a bot.`,
		Format: WebhookResponse{
			Type: WebhookBotEditReviewData{}.Event(),
			Data: WebhookBotEditReviewData{},
		},
		FormatName: "WebhookResponse-WebhookEditBotReviewData",
	}
}

func init() {
	RegisterEvent(WebhookBotEditReviewData{})
}
