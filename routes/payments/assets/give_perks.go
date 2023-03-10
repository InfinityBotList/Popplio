package assets

import (
	"context"
	"errors"
	"fmt"
	"popplio/payments"
	"popplio/state"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type PerkData struct {
	ProductName string `json:"name" validate:"required" msg:"Product name is required."`
	ProductID   string `json:"id" validate:"required" msg:"Product ID is required."`
	For         string `json:"for" validate:"required" msg:"For is required."`
}

// Finds and validates the associated perm for the given payload. ProductID is still needed to determine whats being purchased.
func FindPerks(ctx context.Context, payload PerkData) (*payments.Plan, error) {
	var perk *payments.Plan

	fmt.Println(payload)

	switch payload.ProductID {
	case "premium":
		for _, plan := range payments.Plans {
			fmt.Println(plan.ID, payload.ProductName)
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

func GivePerks(ctx context.Context, userID string, perkData PerkData) error {
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
			Content: "<@" + userID + "> has bought <@" + botID + "> premium for " + strconv.Itoa(perk.TimePeriod) + " hours.",
		})

		if err != nil {
			return errors.New("couldn't send message to mod logs")
		}
	}

	return nil
}