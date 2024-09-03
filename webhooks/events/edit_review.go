package events

import (
	"fmt"
	"popplio/validators"
	"popplio/webhooks/core/events"

	"github.com/disgoorg/disgo/discord"
	"github.com/infinitybotlist/eureka/dovewing/dovetypes"
)

type WebhookEditReviewData struct {
	ReviewID    string                   `json:"review_id" description:"The ID of the review"`
	Stars       events.Changeset[int32]  `json:"stars" description:"The number of stars the auther gave to the review"`
	Content     events.Changeset[string] `json:"content" description:"The content of the review"`
	OwnerReview bool                     `json:"owner_review" description:"Whether or not the review was left by the owner of the entity"`
}

func (v WebhookEditReviewData) TargetTypes() []string {
	return []string{
		"bot",
		"server",
	}
}

func (n WebhookEditReviewData) Event() string {
	return "EDIT_REVIEW"
}

func (n WebhookEditReviewData) Summary() string {
	return "Edit Review"
}

func (n WebhookEditReviewData) Description() string {
	return "This webhook is sent when a user edits an existing review on an entity."
}

func (n WebhookEditReviewData) CreateDiscordEmbed(creator *dovetypes.PlatformUser, targets events.Target) *discord.Embed {
	return &discord.Embed{

		URL: "https://botlist.site/" + targets.GetID(),
		Thumbnail: &discord.EmbedResource{
			URL: targets.GetAvatarURL(),
		},
		Title:       "📝 Review Editted!",
		Description: ":heart: " + creator.DisplayName + " has editted a review for " + targets.GetTargetName(),
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
				Inline: validators.TruePtr,
			},
			{
				Name: "New Content",
				Value: func() string {
					if len(n.Content.New) > 1000 {
						return n.Content.New[:1000] + "..."
					}

					return n.Content.New
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
	events.AddEvent(WebhookEditReviewData{})
}
