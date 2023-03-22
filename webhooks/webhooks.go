package webhooks

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"popplio/state"
	"strings"

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

// Generic because docs
type WebhookResponse struct {
	Creator   *dovewing.DiscordUser `json:"creator"`
	Bot       *dovewing.DiscordUser `json:"bot"`
	CreatedAt int                   `json:"created_at"`
	Type      WebhookType           `json:"type" dynexample:"true" enum:"0,1,2"`

	// The data of the webhook may differ based on its webhook type
	//
	// If the webhook type is WebhookTypeVoteNormal or WebhookTypeVoteTest, the data will be of type WebhookVoteData
	// If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
	Data any `json:"data" dynschema:"true"`
}

type WebhookVoteData struct {
	Votes int `json:"votes"` // The amount of votes the bot received
}

type WebhookNewReviewData struct {
	ReviewID string `json:"review_id"` // The ID of the review
	Content  string `json:"content"`   // The content of the review
}

// Internal structs
type webhookSendState struct {
	Tries int
	URL   string
	Data  []byte
	Sign  string
}

// Actual code

// Validates the webhook
func (w *WebhookResponse) Validate() error {
	var ok bool

	switch w.Type {
	case WebhookTypeVoteNormal, WebhookTypeVoteTest:
		_, ok = w.Data.(WebhookVoteData)
	case WebhookTypeNewReview:
		_, ok = w.Data.(WebhookNewReviewData)
	}

	if !ok {
		return errors.New("invalid webhook data")
	}

	return nil
}

// Returns true if the given url is a valid discord webhook url
func isDiscordURL(url string) bool {
	validPrefixes := []string{
		"https://discordapp.com/api/webhooks/",
		"https://discord.com/api/webhooks/",
		"https://canary.discord.com/api/webhooks/",
		"https://ptb.discord.com/api/webhooks/",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
}

// Creates a discord webhook send
func SendDiscord(url string) error {
	return nil
}

// Creates a custom webhook response, retrying if needed
func SendCustom(d *webhookSendState) error {
	return nil
}

// Creates a webhook response, creating a HMAC signature from request body
func (w *WebhookResponse) Create() error {
	// Validate webhook
	if err := w.Validate(); err != nil {
		return err
	}

	// Fetch the webhook url from db
	var webhookURL string
	err := state.Pool.QueryRow(state.Context, "SELECT webhook_url FROM bots WHERE bot_id = $1", w.Bot.ID).Scan(&webhookURL)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook url")
	}

	// Abort early and call SendDiscord if the webhook url is a discord webhook url
	if isDiscordURL(webhookURL) {
		return SendDiscord(webhookURL)
	}

	var webhookSecret string
	err = state.Pool.QueryRow(state.Context, "SELECT web_auth FROM bots WHERE bot_id = $1", w.Bot.ID).Scan(&webhookSecret)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook secret")
	}

	payload, err := json.Marshal(w)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to marshal webhook payload")
	}

	// Generate HMAC token using token and request body
	h := hmac.New(sha512.New, []byte(webhookSecret))
	h.Write(payload)
	finalToken := hex.EncodeToString(h.Sum(nil))

	return SendCustom(&webhookSendState{
		URL:  webhookURL,
		Sign: finalToken,
		Data: payload,
	})
}

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
		Format: WebhookResponse{
			Type: WebhookTypeVoteTest,
			Data: WebhookVoteData{},
		},
		FormatName: "WebhookResponse-WebhookVoteData",
	})
}
