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
	Votes int  `json:"votes"` // The amount of votes the bot received
	Test  bool `json:"test"`  // Whether the vote was a test vote or not
}

type WebhookBotNewReviewData struct {
	ReviewID string `json:"review_id"` // The ID of the review
	Content  string `json:"content"`   // The content of the review
}

// IMPL
type WebhookResponse struct {
	Creator   *dovewing.DiscordUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	CreatedAt int64                 `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Type      WebhookType           `json:"type" dynexample:"true"`
	Data      any                   `json:"data" dynschema:"true"`
	Entity    any                   `json:"entity" dynschema:"true" description:"The entity that the webhook is about (e.g the bot that was voted for etc)"`
}
