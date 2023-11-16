package events

import (
	"fmt"
	"popplio/webhooks/core/events"

	"github.com/bwmarrin/discordgo"
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

func (n WebhookEditReviewData) CreateHookParams(creator *dovetypes.PlatformUser, targets events.Target) *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				URL: "https://botlist.site/" + targets.GetID(),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: targets.GetAvatarURL(),
				},
				Title:       "📝 Review Editted!",
				Description: ":heart: " + creator.DisplayName + " has editted a review for " + targets.GetTargetName(),
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
	events.RegisterEvent(WebhookEditReviewData{})
}
