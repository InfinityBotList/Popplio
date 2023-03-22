package webhooks

import (
	"github.com/infinitybotlist/dovewing"
)

// v2 webhooks
type WebhookType int

const (
	WebhookTypeVoteNormal WebhookType = iota
	WebhookTypeVoteTest
	WebhookTypeNewReview // To be implemented
)

type WebhookResponse struct {
	Creator   *dovewing.DiscordUser `json:"creator"`
	Bot       *dovewing.DiscordUser `json:"bot"`
	CreatedAt int                   `json:"created_at"`
	Type      WebhookType           `json:"type"`

	// The data of the webhook may differ based on its webhook type
	//
	// If the webhook type is WebhookTypeVoteNormal or WebhookTypeVoteTest, the data will be of type WebhookVoteData
	// If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
	Data any `json:"data"`
}

type WebhookVoteData struct {
	Votes int `json:"votes"` // The amount of votes the bot received
}

type WebhookNewReviewData struct {
	ReviewID string `json:"review_id"` // The ID of the review
	Content  string `json:"content"`   // The content of the review
}

// Actual code
