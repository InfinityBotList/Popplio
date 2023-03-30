// Package bothooks contains the webhook handlers for bots.
//
// A new webhook handler for a different entity such as a team can be created by creating a new folder here
package bothooks

import (
	"errors"
	"popplio/state"
	"popplio/webhooks/events"
	"popplio/webhooks/sender"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	docs "github.com/infinitybotlist/doclib"
	"github.com/infinitybotlist/dovewing"
	"github.com/jackc/pgx/v5/pgtype"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Simple ergonomic webhook builder
type With struct {
	UserID string
	BotID  string
}

type CreateHook struct {
	Type events.WebhookType
	Data events.Data
}

type withCreateHook struct {
	Type   events.WebhookType
	Data   events.Data
	user   *dovewing.DiscordUser
	bot    *dovewing.DiscordUser
	entity sender.WebhookEntity

	valid bool
}

func (c CreateHook) WithCustom(user *dovewing.DiscordUser, bot *dovewing.DiscordUser) *withCreateHook {
	return &withCreateHook{
		Type: c.Type,
		Data: c.Data,
		user: user,
		bot:  bot,
		entity: sender.WebhookEntity{
			EntityID:   bot.ID,
			EntityName: bot.Username,
			DeleteWebhook: func() error {
				_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", bot.ID)

				if err != nil {
					return err
				}

				return nil
			},
		},
	}
}

// Fills in Bot and Creator from IDs
func (c CreateHook) With(with With) withCreateHook {
	bot, err := dovewing.GetDiscordUser(state.Context, with.BotID)

	if err != nil {
		state.Logger.Error(err)
		return withCreateHook{valid: false}
	}

	user, err := dovewing.GetDiscordUser(state.Context, with.UserID)

	if err != nil {
		state.Logger.Error(err)
		return withCreateHook{valid: false}
	}

	state.Logger.Info("Sending webhook for bot " + bot.ID)

	entity := sender.WebhookEntity{
		EntityID:   bot.ID,
		EntityName: bot.Username,
		DeleteWebhook: func() error {
			_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", bot.ID)

			if err != nil {
				return err
			}

			return nil
		},
	}

	return withCreateHook{
		Type:   c.Type,
		Data:   c.Data,
		user:   user,
		bot:    bot,
		entity: entity,
		valid:  true,
	}
}

func (c withCreateHook) Send() error {
	if !c.valid {
		return errors.New("invalid webhook")
	}

	resp := &events.WebhookResponse{
		Creator:   c.user,
		CreatedAt: time.Now().Unix(),
		Type:      c.Type,
		Data:      c.Data,
	}

	// Fetch the webhook url from db
	var webhookURL string
	var webhooksV2 bool
	err := state.Pool.QueryRow(state.Context, "SELECT webhook, webhooks_v2 FROM bots WHERE bot_id = $1", c.bot.ID).Scan(&webhookURL, &webhooksV2)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook url")
	}

	if !webhooksV2 {
		state.Logger.Warn("webhooks v2 is not enabled for this bot, ignoring")
		return nil
	}

	// Validate webhook
	evt, err := resp.Validate()

	if err != nil {
		return err
	}

	resp.Data = resp.Data.SetEntity(c.bot)

	params := evt.CreateHookParams(resp)

	ok, err := sender.SendDiscord(webhookURL, func() error {
		_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", c.bot.ID)

		if err != nil {
			return err
		}

		return nil
	}, params)

	if err != nil {
		state.Logger.Error(err)
		return err
	}

	if ok {
		return nil
	}

	var webhookSecret pgtype.Text
	err = state.Pool.QueryRow(state.Context, "SELECT web_auth FROM bots WHERE bot_id = $1", c.bot.ID).Scan(&webhookSecret)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to fetch webhook secret")
	}

	payload, err := json.Marshal(resp)

	if err != nil {
		state.Logger.Error(err)
		return errors.New("failed to marshal webhook payload")
	}

	return sender.SendCustom(&sender.WebhookSendState{
		Url: webhookURL,
		Sign: sender.Secret{
			Raw: webhookSecret.String,
		},
		Data:   payload,
		UserID: c.user.ID,
		Entity: c.entity,
	})
}

func Setup() {
	go sender.PullPending(sender.WebhookPullPending{
		EntityType: sender.WebhookEntityTypeBot,
		GetEntity: func(id string) (sender.WebhookEntity, error) {
			bot, err := dovewing.GetDiscordUser(state.Context, id)

			if err != nil {
				return sender.WebhookEntity{}, err
			}

			return sender.WebhookEntity{
				EntityID:   bot.ID,
				EntityName: bot.Username,
				EntityType: sender.WebhookEntityTypeBot,
				DeleteWebhook: func() error {
					_, err := state.Pool.Exec(state.Context, "UPDATE bots SET webhook = NULL WHERE bot_id = $1", bot.ID)

					if err != nil {
						return err
					}

					return nil
				},
			}, nil
		},
	})

	events.RegisteredEvents.AddEvent(events.WebhookTypeBotVote, events.EventData{
		Docs: &docs.WebhookDoc{
			Name:    "NewBotVote",
			Summary: "New Bot Vote",
			Tags: []string{
				"Webhooks",
			},
			Description: `This webhook is sent when a user votes for a bot.
	
	The data of the webhook may differ based on its webhook type
	
	If the webhook type is WebhookTypeVote, the data will be of type WebhookVoteData`,
			Format: events.WebhookResponse{
				Type: events.WebhookTypeBotVote,
				Data: events.WebhookBotVoteData{},
			},
			FormatName: "WebhookResponse-WebhookBotVoteData",
		},
		Format: events.WebhookBotVoteData{},
		CreateHookParams: func(w *events.WebhookResponse) *discordgo.WebhookParams {
			voteData := w.Data.(events.WebhookBotVoteData)

			return &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{
					{
						URL: "https://botlist.site/" + voteData.Bot.ID,
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL: voteData.Bot.Avatar,
						},
						Title:       "🎉 Vote Count Updated!",
						Description: ":heart:" + w.Creator.Username + "#" + w.Creator.Discriminator + " has voted for " + voteData.Bot.Username,
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
								Value:  "[View " + voteData.Bot.Username + "](https://botlist.site/" + voteData.Bot.ID + ")",
								Inline: true,
							},
							{
								Name:   "Vote Page",
								Value:  "[Vote for " + voteData.Bot.Username + "](https://botlist.site/" + voteData.Bot.ID + "/vote)",
								Inline: true,
							},
						},
					},
				},
			}
		},
	})

	events.RegisteredEvents.AddEvent(events.WebhookTypeBotNewReview, events.EventData{
		Docs: &docs.WebhookDoc{
			Name:    "NewBotReview",
			Summary: "New Bot Review",
			Tags: []string{
				"Webhooks",
			},
			Description: `This webhook is sent when a user creates a new review on a bot.
	
	The data of the webhook may differ based on its webhook type
	
	If the webhook type is WebhookTypeNewReview, the data will be of type WebhookNewReviewData
	`,
			Format: events.WebhookResponse{
				Type: events.WebhookTypeBotNewReview,
				Data: events.WebhookBotNewReviewData{},
			},
			FormatName: "WebhookResponse-WebhookNewReviewData",
		},
		Format: events.WebhookBotNewReviewData{},
		CreateHookParams: func(w *events.WebhookResponse) *discordgo.WebhookParams {
			reviewData := w.Data.(events.WebhookBotNewReviewData)

			return &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{
					{
						URL: "https://botlist.site/" + reviewData.Bot.ID,
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL: reviewData.Bot.Avatar,
						},
						Title:       "📝 New Review!",
						Description: ":heart:" + w.Creator.Username + "#" + w.Creator.Discriminator + " has left a review for " + reviewData.Bot.Username,
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
								Value:  "[View " + reviewData.Bot.Username + "](https://botlist.site/" + reviewData.Bot.ID + ")",
								Inline: true,
							},
						},
					},
				},
			}
		},
	})
}
