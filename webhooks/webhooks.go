package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"math/rand"
	"net/http"
	"popplio/notifications"
	"popplio/state"
	"popplio/types"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"
	"github.com/infinitybotlist/eureka/crypto"
)

// v2 webhooks
const MaxWebhookTries = 5

type WebhookSaveState int

const (
	WebhookSaveStatePending WebhookSaveState = iota
	WebhookSaveStateSuccess
	WebhookSaveStateFailed
	WebhookSaveStateRemoved
)

type WebhookType int

const (
	WebhookTypeUnknown WebhookType = iota
	WebhookTypeVote
	WebhookTypeNewReview // To be implemented
)

// Generic because docs
type WebhookResponse struct {
	Creator   *dovewing.DiscordUser `json:"creator" description:"The user who created the action/event (e.g voted for the bot or made a review)"`
	Bot       *dovewing.DiscordUser `json:"bot" description:"The bot that the action/event was performed on"`
	CreatedAt int                   `json:"created_at" description:"The time in *seconds* (unix epoch) of when the action/event was performed"`
	Type      WebhookType           `json:"type" dynexample:"true" enum:"0,1,2"`

	// The data of the webhook may differ based on its webhook type
	//
	// If the webhook type is WebhookTypeVote, the data will be of type WebhookVoteData
	// If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
	Data any `json:"data" dynschema:"true"`
}

type WebhookVoteData struct {
	Votes int  `json:"votes"` // The amount of votes the bot received
	Test  bool `json:"test"`  // Whether the vote was a test vote or not
}

type WebhookNewReviewData struct {
	ReviewID string `json:"review_id"` // The ID of the review
	Content  string `json:"content"`   // The content of the review
}

// Internal structs
type webhookSendState struct {
	tries int
	url   string
	data  []byte
	sign  string

	// Intentionally bad to trigger 401 check
	badIntent bool

	// Automatically set fields
	logID string
}

func (st *webhookSendState) cancelSend(saveState WebhookSaveState) {
	st.tries = MaxWebhookTries

	state.Logger.Warnf("Cancelling webhook send for %s", st.logID)

	_, err := state.Pool.Exec(state.Context, "UPDATE webhooks SET state = $1 WHERE log_id = $2", saveState, st.logID)

	if err != nil {
		state.Logger.Errorf("Failed to update webhook state for %s: %s", st.logID, err.Error())
	}
}

// Actual code

// Simple ergonomic webhook builder
type WebhookWithBot struct {
	UserID string
	BotID  string
}

// Fills in Bot and Creator from IDs
func (w *WebhookResponse) WithBot(with WebhookWithBot) *WebhookResponse {
	bot, err := dovewing.GetDiscordUser(state.Context, with.BotID)

	if err != nil {
		state.Logger.Error(err)
		return w
	}

	user, err := dovewing.GetDiscordUser(state.Context, with.UserID)

	if err != nil {
		state.Logger.Error(err)
		return w
	}

	state.Logger.Info("Sending webhook for bot " + bot.ID)

	w.Bot = bot
	w.Creator = user

	return w
}

// Validates the webhook
func (w *WebhookResponse) Validate() error {
	if w.Creator == nil || w.Bot == nil {
		return errors.New("invalid webhook creator or bot data")
	}

	var ok bool

	switch w.Type {
	case WebhookTypeVote:
		_, ok = w.Data.(WebhookVoteData)
	case WebhookTypeNewReview:
		_, ok = w.Data.(WebhookNewReviewData)
	}

	if !ok {
		return errors.New("invalid webhook data")
	}

	return nil
}

func isDiscordAPIURL(url string) (bool, string) {
	validPrefixes := []string{
		"https://discordapp.com/",
		"https://discord.com/",
		"https://canary.discord.com/",
		"https://ptb.discord.com/",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true, prefix
		}
	}

	return false, ""
}

// Creates a discord webhook send
func (w *WebhookResponse) sendDiscord(url string, prefix string) error {
	// Remove out prefix
	url = state.Config.Meta.PopplioProxy + "/" + strings.TrimPrefix(url, prefix)

	if !strings.Contains(url, "/webhooks/") {
		return errors.New("invalid discord webhook url")
	}

	var data *discordgo.WebhookParams

	switch w.Type {
	case WebhookTypeVote:
		voteData := w.Data.(WebhookVoteData)

		data = &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL: "https://botlist.site/" + w.Bot.ID,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: w.Bot.Avatar,
					},
					Title:       "🎉 Vote Count Updated!",
					Description: ":heart:" + w.Creator.Username + "#" + w.Creator.Discriminator + " has voted for " + w.Bot.Username,
					Color:       0x8A6BFD,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Vote Count:",
							Value:  strconv.Itoa(int(voteData.Votes)),
							Inline: true,
						},
						{
							Name:   "User ID:",
							Value:  w.Creator.ID,
							Inline: true,
						},
						{
							Name:   "Vote Page",
							Value:  "[View " + w.Bot.Username + "](https://botlist.site/" + w.Bot.ID + ")",
							Inline: true,
						},
						{
							Name:   "Vote Page",
							Value:  "[Vote for " + w.Bot.Username + "](https://botlist.site/" + w.Bot.ID + "/vote)",
							Inline: true,
						},
					},
				},
			},
		}
	case WebhookTypeNewReview:
		reviewData := w.Data.(WebhookNewReviewData)

		data = &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL: "https://botlist.site/" + w.Bot.ID,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: w.Bot.Avatar,
					},
					Title:       "📝 New Review!",
					Description: ":heart:" + w.Creator.Username + "#" + w.Creator.Discriminator + " has left a review for " + w.Bot.Username,
					Color:       0x8A6BFD,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Review ID:",
							Value:  reviewData.ReviewID,
							Inline: true,
						},
						{
							Name:   "User ID:",
							Value:  w.Creator.ID,
							Inline: true,
						},
						{
							Name: "Review Content:",
							Value: func() string {
								if len(reviewData.Content) > 1000 {
									return reviewData.Content[:1000] + "..."
								}

								return reviewData.Content
							}(),
							Inline: true,
						},
						{
							Name:   "Review Page",
							Value:  "[View " + w.Bot.Username + "](https://botlist.site/" + w.Bot.ID + ")",
							Inline: true,
						},
					},
				},
			},
		}
	}

	payload, err := json.Marshal(data)

	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))

	if err != nil {
		return err
	}

	for _, code := range []int{404, 401, 403, 410} {
		if resp.StatusCode == code {
			// Remove webhook URL
			_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", w.Bot.ID)

			if err != nil {
				state.Logger.Error(err)
			}

			return errors.New("bad webhook url (removed from db)")
		}
	}

	state.Logger.With(
		"botId", w.Bot.ID,
		"statusCode", resp.StatusCode,
	).Info("sent discord webhook")

	return nil
}

// Creates a custom webhook response, retrying if needed
func (w *WebhookResponse) sendCustom(d *webhookSendState) error {
	d.tries++

	if d.logID == "" {
		// Add to webhook logs for automatic retry
		var logID string
		err := state.Pool.QueryRow(state.Context, "INSERT INTO webhook_logs (bot_id, user_id, url, data, sign, bad_intent) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", w.Bot.ID, w.Creator.ID, d.url, d.data, d.sign, d.badIntent).Scan(&logID)

		if err != nil {
			return err
		}

		d.logID = logID
	} else {
		// Update webhook logs
		_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET tries = tries + 1 WHERE id = $1", d.logID)

		if err != nil {
			return err
		}
	}

	if d.tries > MaxWebhookTries {
		_, err := state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.logID, WebhookSaveStateFailed)

		if err != nil {
			return err
		}

		return errors.New("too many tries")
	}

	state.Logger.With(
		"botId", w.Bot.ID,
		"userId", w.Creator.ID,
		"tries", d.tries,
	)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(state.Context, "POST", d.url, bytes.NewReader(d.data))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Popplio/v7.0.0 (https://infinitybots.gg)")
	req.Header.Set("X-Webhook-Signature", d.sign)
	req.Header.Set("X-Webhook-Protocol", "splashtail")

	resp, err := client.Do(req)

	if err != nil {
		state.Logger.Error(err)
		time.Sleep(5 * time.Minute)
		return w.sendCustom(d)
	}

	switch {
	case resp.StatusCode >= 429:
		// Retry after
		retryAfter := resp.Header.Get("Retry-After")

		if retryAfter == "" {
			time.Sleep(5 * time.Minute)
			return w.sendCustom(d)
		}

		retryAfterInt, err := strconv.Atoi(retryAfter)

		if err != nil {
			state.Logger.With(
				"retryAfter", retryAfter,
			).Error(err)
			time.Sleep(5 * time.Minute)
			return w.sendCustom(d)
		}

		time.Sleep(time.Duration(retryAfterInt+d.tries^2+15) * time.Second)
		return w.sendCustom(d)

	case resp.StatusCode == 404 || resp.StatusCode == 410:
		// Remove from DB
		d.cancelSend(WebhookSaveStateFailed)
		_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", w.Bot.ID)

		if err != nil {
			state.Logger.Error(err)
			return err
		}

		// Remove from webhook logs
		_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.logID, WebhookSaveStateRemoved)

		if err != nil {
			state.Logger.Error(err)
			return err
		}

		return errors.New("webhook returned not found thus removing it from the database")

	case resp.StatusCode == 401 || resp.StatusCode == 403:
		if d.badIntent {
			// webhook auth is invalid as intended,
			d.cancelSend(WebhookSaveStateSuccess)

			return nil
		} else {
			// webhook auth is invalid, return error
			err = notifications.PushNotification(w.Creator.ID, types.Notification{
				Type:    "info",
				Message: "This webhook does not properly handle authentication at this time.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error(err)
			}

			return errors.New("webhook auth error")
		}

	case resp.StatusCode >= 400:
		time.Sleep(10 * time.Minute)
		return w.sendCustom(d)

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if d.badIntent {
			d.cancelSend(WebhookSaveStateRemoved)

			err = notifications.PushNotification(w.Creator.ID, types.Notification{
				Type:    "info",
				Message: "This webhook does not properly handle authentication at this time.",
				Title:   "Webhook Auth Error",
			})

			if err != nil {
				state.Logger.Error(err)
			}

			// Remove webhook, it doesn't validate auth at all
			_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", w.Bot.ID)

			if err != nil {
				state.Logger.Error(err)
				return errors.New("webhook failed to validate auth and failed to remove webhook from db")
			}

			// Remove from webhook logs
			_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.logID, WebhookSaveStateRemoved)

			if err != nil {
				state.Logger.Error(err)
				return errors.New("webhook failed to validate auth and failed to remove webhook from logdb")
			}

			return errors.New("webhook failed to validate auth thus removing it from the database")
		} else {
			d.cancelSend(WebhookSaveStateSuccess)
		}
	case resp.StatusCode >= 500:
		// Give 15 minutes to recover
		time.Sleep(15 * time.Minute)
		return w.sendCustom(d)
	}

	_, err = state.Pool.Exec(state.Context, "UPDATE webhook_logs SET state = $2 WHERE id = $1", d.logID, WebhookSaveStateSuccess)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	err = notifications.PushNotification(w.Creator.ID, types.Notification{
		Type:    "success",
		Message: "Successfully notified bot " + w.Bot.Username + " of this action.",
		Title:   "Webhook Send Successful!",
	})

	if err != nil {
		state.Logger.Error(err)
	}

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
	err := state.Pool.QueryRow(state.Context, "SELECT webhook FROM bots WHERE bot_id = $1", w.Bot.ID).Scan(&webhookURL)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook url")
	}

	// Abort early and call SendDiscord if the webhook url is a discord webhook url
	if ok, prefix := isDiscordAPIURL(webhookURL); ok {
		return w.sendDiscord(webhookURL, prefix)
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

	// Check if random number is less than 0.5 for fake
	if rand.Float64() < 0.5 {
		go func() {
			badD := &webhookSendState{
				tries:     3,
				badIntent: true,
				sign:      crypto.RandString(128),
				url:       webhookURL,
				data:      payload,
			}

			// Retry with bad intent
			w.sendCustom(badD)
		}()
	}

	return w.sendCustom(&webhookSendState{
		url:  webhookURL,
		sign: finalToken,
		data: payload,
	})
}

// Resend pending hooks
func resendPending() {
	// Fetch every pending webhook from webhook_logs
	rows, err := state.Pool.Query(state.Context, "SELECT id, bot_id, user_id, url, data, sign, bad_intent, tries FROM webhook_logs WHERE state = $1", WebhookSaveStatePending)

	if err != nil {
		state.Logger.Error(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			botId     string
			userId    string
			url       string
			data      []byte
			sign      string
			badIntent bool
			tries     int
		)

		err := rows.Scan(&id, &botId, &userId, &url, &data, &sign, &badIntent, &tries)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		// Create webhook
		bot, err := dovewing.GetDiscordUser(state.Context, botId)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		user, err := dovewing.GetDiscordUser(state.Context, userId)

		if err != nil {
			state.Logger.Error(err)
			continue
		}

		w := &WebhookResponse{
			Bot:     bot,
			Creator: user,
		}

		// Send webhook
		err = w.sendCustom(&webhookSendState{
			url:       url,
			sign:      sign,
			data:      data,
			badIntent: badIntent,
			logID:     id,
			tries:     tries,
		})

		if err != nil {
			state.Logger.Error(err)
		}
	}
}

// Setup code
func Setup() {
	// Create webhook_logs
	_, err := state.Pool.Exec(state.Context, `CREATE TABLE IF NOT EXISTS webhook_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
		bot_id TEXT NOT NULL REFERENCES bots(bot_id), 
		user_id TEXT NOT NULL REFERENCES users(user_id), 
		url TEXT NOT NULL, 
		data JSONB NOT NULL, 
		sign TEXT NOT NULL, 
		bad_intent BOOLEAN NOT NULL, 
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
		state INTEGER NOT NULL DEFAULT 0, 
		tries INTEGER NOT NULL DEFAULT 0, 
		last_try TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	if err != nil {
		panic(err)
	}

	go resendPending()

	docs.AddTag(
		"Webhooks",
		"Webhooks are a way to receive events from Infinity Bot List in real time. You can use webhooks to receive events such as new votes, new reviews, and more.",
	)

	docs.AddWebhook(&docs.WebhookDoc{
		Name:    "NewBotVote",
		Summary: "New Bot Vote",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user votes for a bot.

The data of the webhook may differ based on its webhook type

If the webhook type is WebhookTypeVote, the data will be of type WebhookVoteData`,
		Format: WebhookResponse{
			Type: WebhookTypeVote,
			Data: WebhookVoteData{},
		},
		FormatName: "WebhookResponse-WebhookVoteData",
	})

	docs.AddWebhook(&docs.WebhookDoc{
		Name:    "NewBotReview",
		Summary: "New Bot Review",
		Tags: []string{
			"Webhooks",
		},
		Description: `This webhook is sent when a user creates a new review on a bot.

The data of the webhook may differ based on its webhook type

If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
`,
		Format: WebhookResponse{
			Type: WebhookTypeNewReview,
			Data: WebhookNewReviewData{},
		},
		FormatName: "WebhookResponse-WebhookNewReviewData",
	})
}
