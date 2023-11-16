package events

import (
	"fmt"
	"popplio/webhooks/core/events"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookNewReviewData struct {
	ReviewID    string `json:"review_id" description:"The ID of the review"`
	Content     string `json:"content" description:"The content of the review"`
	Stars       int32  `json:"stars" description:"The number of stars the auther gave to the review"`
	OwnerReview bool   `json:"owner_review" description:"Whether or not the review was left by the owner of the entity"`
}

func (v WebhookNewReviewData) TargetTypes() []string {
	return []string{
		"bot",
		"server",
	}
}

func (n WebhookNewReviewData) Event() string {
	return "NEW_REVIEW"
}

func (n WebhookNewReviewData) Summary() string {
	return "New Review"
}

func (n WebhookNewReviewData) Description() string {
	return "This webhook is sent when a user creates a new review on an entity."
}

func (n WebhookNewReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets events.Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.GetID(),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.GetAvatarURL(),
				},
				Title:       "📝 New Review!",
				Description: ":heart: " + creator.DisplayName + " has left a review for " + targets.GetTargetName(),
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
						Name: "Owner Review",
						Value: func() string {
							if n.OwnerReview {
								return "Yes"
							}

							return "No"
						}(),
					},
					{
						Name:   "Review Page",
						Value:  targets.GetViewLink(),
						Inline: true,
					},
				},
			},
		},
	}
}

func init() {
	events.RegisterEvent(WebhookNewReviewData{})
}
