package events

import (
	"fmt"
	"popplio/webhooks/core/events"

	"github.com/bwmarrin/discordgo"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookDeleteReviewData struct {
	ReviewID    string `json:"review_id" description:"The ID of the review"`
	Content     string `json:"content" description:"The content of the review at time of deletion"`
	Stars       int32  `json:"stars" description:"The number of stars the auther gave to the review at time of deletion"`
	OwnerReview bool   `json:"owner_review" description:"Whether or not the review was an owner review"`
}

func (v WebhookDeleteReviewData) TargetTypes() []string {
	return []string{
		"bot",
		"server",
	}
}

func (n WebhookDeleteReviewData) Event() string {
	return "DELETE_REVIEW"
}

func (n WebhookDeleteReviewData) Summary() string {
	return "Delete Review"
}

func (n WebhookDeleteReviewData) Description() string {
	return "This webhook is sent when a user delete their review on an entity."
}

func (n WebhookDeleteReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets events.Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.GetID(),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.GetAvatarURL(),
				},
				Title:       ":x: Bot Review Deleted!",
				Description: creator.DisplayName + " has deleted his review for " + targets.GetTargetName(),
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
						Value:  targets.GetViewLink(),
						Inline: true,
					},
					{
						Name:   "Owner Review",
						Value:  fmt.Sprintf("%t", n.OwnerReview),
						Inline: true,
					},
				},
			},
		},
	}
}

func init() {
	events.AddEvent(WebhookDeleteReviewData{})
}
