package events

import (
	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

const WebhookTypeServerEditReview WebhookType = "SERVER_EDIT_REVIEW"

type WebhookServerEditReviewData struct {
	ReviewID string            `json:"review_id" description:"The ID of the review"`
	Content  Changeset[string] `json:"content" description:"The content of the review"`
}

func (v WebhookServerEditReviewData) TargetType() string {
	return "server"
}

func (n WebhookServerEditReviewData) Event() WebhookType {
	return WebhookTypeServerEditReview
}

func (n WebhookServerEditReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.Server.ID,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.Server.Avatar,
				},
				Title:       "📝 Review Editted!",
				Description: ":heart: " + creator.DisplayName + " has editted a review for server" + targets.Server.Name,
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
						Value:  "[View " + targets.Server.Name + "](https://botlist.site/" + targets.Server.ID + ")",
						Inline: true,
					},
				},
			},
		},
	}
}

func (n WebhookServerEditReviewData) Docs() *docs.WebhookDoc {
	return &docs.WebhookDoc{
		Name:    "EditServerReview",
		Summary: "Edit Server Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user edits an existing review on a server.`,
		Format: WebhookResponse{
			Type: WebhookServerEditReviewData{}.Event(),
			Data: WebhookServerEditReviewData{},
		},
		FormatName: "WebhookResponse-WebhookEditServerReviewData",
	}
}

func init() {
	RegisterEvent(WebhookServerEditReviewData{})
}
