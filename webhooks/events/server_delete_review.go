package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeServerDeleteReview WebhookType = "SERVER_DELETE_REVIEW"

type WebhookServerDeleteReviewData struct {
	ReviewID string `json:"review_id" description:"The ID of the review"`
	Content  string `json:"content" description:"The content of the review at time of deletion"`
	Stars    int32  `json:"stars" description:"The number of stars the auther gave to the review at time of deletion"`
}

func (v WebhookServerDeleteReviewData) TargetType() string {
	return "server"
}

func (n WebhookServerDeleteReviewData) Event() WebhookType {
	return WebhookTypeServerDeleteReview
}

func (n WebhookServerDeleteReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Server.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Server.Avatar,
				},
				Title:       ":x: Server Review Deleted!",
				Description: creator.DisplayName + " has deleted his review for server " + targets.Server.Name,
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
						Value:  "[View " + targets.Server.Name + "](https://botlist.site/" + targets.Server.ID + ")",
						Inline: true,
					},
				},
			},
		},
	}
}

func (n WebhookServerDeleteReviewData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "DeleteServerReview",
		Summary: "Delete Server Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user deletes their review on a server.`,
		Format: WebhookResponse{
			Type: WebhookBotDeleteReviewData{}.Event(),
			Data: WebhookBotDeleteReviewData{},
		},
		FormatName: "WebhookResponse-WebhookDeleteServerReviewData",
	}
}

func init() {
	RegisterEvent(WebhookServerDeleteReviewData{})
}
