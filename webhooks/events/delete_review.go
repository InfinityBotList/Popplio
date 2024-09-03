package events

import (
	"fmt"
	"popplio/validators"
	"popplio/webhooks/core/events"

	"github.com/disgoorg/disgo/discord"
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

func (n WebhookDeleteReviewData) CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets events.Target) *discord.Embed {
	return &discord.Embed{
		URL: "https://botlist.site/" + targets.GetID(),
		Thumbnail: &discord.EmbedResource{
			URL: targets.GetAvatarURL(),
		},
		Title:       ":x: Bot Review Deleted!",
		Description: creator.DisplayName + " has deleted his review for " + targets.GetTargetName(),
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
				Name: "Content",
				Value: func() string {
					if len(n.Content) > 1000 {
						return n.Content[:1000] + "..."
					}

					return n.Content
				}(),
				Inline: validators.TruePtr,
			},
			{
				Name:   "Review Page",
				Value:  targets.GetViewLink(),
				Inline: validators.TruePtr,
			},
			{
				Name:   "Owner Review",
				Value:  fmt.Sprintf("%t", n.OwnerReview),
				Inline: validators.TruePtr,
			},
		},
	}
}

func init() {
	events.AddEvent(WebhookDeleteReviewData{})
}
