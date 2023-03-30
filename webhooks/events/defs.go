package events

import "github.com/infinitybotlist/dovewing"

type WebhookType string

const (
	// Bot Events

	WebhookTypeBotVote      WebhookType = "BOT_VOTE"
	WebhookTypeBotNewReview WebhookType = "BOT_NEW_REVIEW"
)

// Bot events
type WebhookBotVoteData struct {
	Bot   *dovewing.DiscordUser `json:"bot"`   // The bot that was voted for
	Votes int                   `json:"votes"` // The amount of votes the bot received
	Test  bool                  `json:"test"`  // Whether the vote was a test vote or not
}

type WebhookBotNewReviewData struct {
	Bot      *dovewing.DiscordUser `json:"bot"`       // The bot that was voted for
	ReviewID string                `json:"review_id"` // The ID of the review
	Content  string                `json:"content"`   // The content of the review
}

// IMPL
type WebhookResponse struct {
	Creator   *dovewing.DiscordUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	CreatedAt int64                 `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Type      WebhookType           `json:"type" dynexample:"true"`

	// The data of the webhook may differ based on its webhook type
	//
	// If the webhook type is WebhookTypeVote, the data will be of type WebhookVoteData
	// If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
	Data any `json:"data" dynschema:"true"`
}
