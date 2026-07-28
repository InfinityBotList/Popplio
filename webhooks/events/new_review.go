package events

import (
	"fmt"
	"popplio/validators"
	"popplio/webhooks/core/events"

	"github.com/disgoorg/disgo/discord"
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

func (n WebhookNewReviewData) CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets events.Target) *discord.Embed {
	return &discord.Embed{
		URL: "https://botlist.site/" + targets.GetID(),
		Thumbnail: &discord.EmbedResource{
			URL: targets.GetAvatarURL(),
		},
		Title:       "📝 New Review!",
		Description: ":heart: " + creator.DisplayName + " has left a review for " + targets.GetTargetName(),
		Color:       0x8A6BFD,
		Fields: []discord.EmbedField{
			{
				Name:   "Review ID",
				Value:  n.ReviewID,
				Inline: validators.TruePtr,
			},
			{
				Name:   "User ID",
				Value:  creator.ID,
				Inline: validators.TruePtr,
			},
			{
				Name:   "Stars",
				Value:  fmt.Sprintf("%d/5", n.Stars),
				Inline: validators.TruePtr,
			},
			{
				Name: "Review Content",
				Value: func() string {
					if len(n.Content) > 1000 {
						return n.Content[:1000] + "..."
					}

					return n.Content
				}(),
				Inline: validators.TruePtr,
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
				Inline: validators.TruePtr,
			},
		},
	}
}

func init() {
	events.AddEvent(WebhookNewReviewData{})
}
