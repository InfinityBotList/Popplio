package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeBotDeleteReview WebhookType = "BOT_DELETE_REVIEW"

type WebhookBotDeleteReviewData struct {
	ReviewID string `json:"review_id" description:"The ID of the review"`
	Content  string `json:"content" description:"The content of the review at time of deletion"`
	Stars    int32  `json:"stars" description:"The number of stars the auther gave to the review at time of deletion"`
}

func (v WebhookBotDeleteReviewData) TargetType() string {
	return "bot"
}

func (n WebhookBotDeleteReviewData) Event() WebhookType {
	return WebhookTypeBotDeleteReview
}

func (n WebhookBotDeleteReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Bot.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Bot.Avatar,
				},
				Title:       ":x: Bot Review Deleted!",
				Description: creator.DisplayName + " has deleted his review for bot " + targets.Bot.Username,
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
						Name:   "Stars",
						Value:  fmt.Sprintf("%d/5", n.Stars),
						Inline: true,
					},
					{
						Name: "Content",
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

func (n WebhookBotDeleteReviewData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "DeleteBotReview",
		Summary: "Delete Bot Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user deletes their review on a bot.`,
		Format: WebhookResponse{
			Type: WebhookBotDeleteReviewData{}.Event(),
			Data: WebhookBotDeleteReviewData{},
		},
		FormatName: "WebhookResponse-WebhookDeleteBotReviewData",
	}
}

func init() {
	RegisterEvent(WebhookBotDeleteReviewData{})
}
