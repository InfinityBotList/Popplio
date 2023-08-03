package assets

import (
	"context"
	"errors"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type CreatePerkData struct {
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
}

type RedirectUser struct {
	URL string `json:"url"`
}

func (c CreatePerkData) Parse(userID string) PerkData {
	return PerkData{
		UserID:      userID,
		ProductName: c.ProductName,
		ProductID:   c.ProductID,
		For:         c.For,
	}
}

type PerkData struct {
	UserID      string `json:"user_id" validate:"required" msg:"Internal error: endpoint must fill in UserID. Please contact support."`
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
}

// Finds and validates the associated perm for the given payload. ProductID is still needed to determine whats being purchased.
func FindPerks(ctx context.Context, payload PerkData) (*types.PaymentPlan, error) {
	var perk *types.PaymentPlan

	state.Logger.Info(payload)

	if payload.UserID == "" {
		return nil, errors.New("internal error: user id is required")
	}

	err := validators.StagingCheckSensitive(ctx, payload.UserID)

	if err != nil {
		return nil, err
	}

	switch payload.ProductID {
	case "premium":
		for _, plan := range Plans {
			state.Logger.Info(plan.ID, payload.ProductName)
			if plan.ID == payload.ProductName {
				// Ensure the bot associated with For exists
				var count int64

				err := state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM bots WHERE bot_id = $1", payload.For).Scan(&count)

				if err != nil {
					return nil, errors.New("our database broke, please try again later")
				}

				if count == 0 {
					return nil, errors.New("bot id is invalid")
				}

				var typeStr string
				var premium bool

				err = state.Pool.QueryRow(ctx, "SELECT type, premium FROM bots WHERE bot_id = $1", payload.For).Scan(&typeStr, &premium)

				if err != nil {
					return nil, errors.New("our database broke, please try again later")
				}

				if typeStr != "approved" && typeStr != "certified" {
					return nil, errors.New("bot is not approved or certified")
				}

				if premium {
					return nil, errors.New("bot is already premium")
				}

				perk = &plan

				break
			}
		}
	default:
		return nil, errors.New("invalid product id")
	}

	if perk == nil {
		return nil, errors.New("product not found")
	}

	return perk, nil
}

func GivePerks(ctx context.Context, perkData PerkData) error {
	err := validators.StagingCheckSensitive(ctx, perkData.UserID)

	if err != nil {
		return err
	}

	perk, err := FindPerks(ctx, perkData)

	if err != nil {
		return err
	}

	// Check if the user has already purchased this perk, if not give it to them
	switch perkData.ProductID {
	case "premium":
		var botID = perkData.For

		_, err = state.Pool.Exec(ctx,
			"UPDATE bots SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE bot_id = $2",
			perk.TimePeriod,
			botID,
		)

		if err != nil {
			return errors.New("our database broke, please try again later")
		}

		_, err = state.Discord.ChannelMessageSendComplex(state.Config.Channels.ModLogs, &discordgo.MessageSend{
			Content: "<@" + perkData.UserID + "> has bought <@" + botID + "> premium for " + strconv.Itoa(perk.TimePeriod) + " hours.",
		})

		if err != nil {
			return errors.New("couldn't send message to mod logs")
		}
	}

	return nil
}
