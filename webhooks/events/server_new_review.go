package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeServerNewReview WebhookType = "SERVER_NEW_REVIEW"

type WebhookServerNewReviewData struct {
	ReviewID string `json:"review_id" description:"The ID of the review"`
	Content  string `json:"content" description:"The content of the review"`
}

func (v WebhookServerNewReviewData) TargetType() string {
	return "server"
}

func (n WebhookServerNewReviewData) Event() WebhookType {
	return WebhookTypeServerNewReview
}

func (n WebhookServerNewReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Server.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Server.Avatar,
				},
				Title:       "📝 New Review!",
				Description: ":heart: " + creator.DisplayName + " has left a review for server " + targets.Server.Name,
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
						Value:  "[View " + targets.Server.Name + "](https://botlist.site/" + targets.Server.ID + ")",
						Inline: true,
					},
				},
			},
		},
	}
}

func (n WebhookServerNewReviewData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "NewServerReview",
		Summary: "New Server Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user creates a new review on a server.`,
		Format: WebhookResponse{
			Type: WebhookServerNewReviewData{}.Event(),
			Data: WebhookServerNewReviewData{},
		},
		FormatName: "WebhookResponse-WebhookServerNewReviewData",
	}
}

func init() {
	RegisterEvent(WebhookServerNewReviewData{})
}
