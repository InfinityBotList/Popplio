package webhooks

import (
	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"
)

// v2 webhooks
type WebhookType int

const (
	WebhookTypeVoteNormal WebhookType = iota
	WebhookTypeVoteTest
	WebhookTypeNewReview // To be implemented
)

type WebhookResponse[T any] struct {
	Creator   *dovewing.DiscordUser `json:"creator"`
	Bot       *dovewing.DiscordUser `json:"bot"`
	CreatedAt int                   `json:"created_at"`
	Type      WebhookType           `json:"type" dynexample:"true" enum:"0,1,2"`

	// The data of the webhook may differ based on its webhook type
	//
	// If the webhook type is WebhookTypeVoteNormal or WebhookTypeVoteTest, the data will be of type WebhookVoteData
	// If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
	Data T `json:"data"`
}

type WebhookVoteData struct {
	Votes int `json:"votes"` // The amount of votes the bot received
}

type WebhookNewReviewData struct {
	ReviewID string `json:"review_id"` // The ID of the review
	Content  string `json:"content"`   // The content of the review
}

// Actual code

// Setup code
func Setup() {
	docs.AddTag(
		"Webhooks",
		"Webhooks are a way to receive events from Infinity Bot List in real time. You can use webhooks to receive events such as new votes, new reviews, and more.",
	)

	docs.AddWebhook(&docs.WebhookDoc{
		Name:    "Vote",
		Summary: "New Bot Vote",
		Tags: []string{
			"Votes",
			"Webhooks",
		},
		Description: `This webhook is sent when a user votes for a bot.

The data of the webhook may differ based on its webhook type

If the webhook type is WebhookTypeVoteNormal or WebhookTypeVoteTest, the data will be of type WebhookVoteData as shown below:
`,
		Format: WebhookResponse[WebhookVoteData]{
			Type: WebhookTypeVoteTest,
		},
		FormatName: "WebhookResponse-WebhookVoteData",
	})
}
