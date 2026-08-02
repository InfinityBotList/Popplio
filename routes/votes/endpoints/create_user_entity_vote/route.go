// Package create_user_entity_vote implements PUT
// /users/{uid}/{target_type}/{target_id}/votes — "Create Entity Vote".
//
// Creates a vote for an entity. Returns 204 on success. Note that for
// compatibility, a trailing 's' is removed
package create_user_entity_vote

import (
	"fmt"
	"net/http"
	"popplio/api/resp"
	"strconv"

	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/votes"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/events"

	"github.com/disgoorg/disgo/discord"
	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Entity Vote",
		Description: "Creates a vote for an entity. Returns 204 on success. Note that for compatibility, a trailing 's' is removed",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "upvote",
				Description: "Whether or not to upvote the entity. Must be either true or false",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	// Check if upvote query parameter is valid
	upvoteStr := r.URL.Query().Get("upvote")

	if upvoteStr != "true" && upvoteStr != "false" {
		return resp.BadRequest("upvote must be either `true` or `false`")
	}

	upvote := upvoteStr == "true"

	// Check if user is allowed to even make a vote right now.
	var voteBanned bool

	err := state.Pool.QueryRow(d.Context, "SELECT vote_banned FROM users WHERE user_id = $1", d.Auth.ID).Scan(&voteBanned)

	if err != nil {
		state.Logger.Error("Failed to check if user is vote banned", zap.Error(err), zap.String("userId", d.Auth.ID))
		return resp.BadRequest("Error checking if user is vote banned.")
	}

	if voteBanned {
		return resp.Forbidden("You are banned from voting right now! Contact support if you think this is a mistake")
	}

	// Create a new entity vote
	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.ErrBody("Failed to create transaction [put_user_entity_votes]", "Failed to create transaction.", err)
	}

	defer tx.Rollback(d.Context)

	entityInfo, err := votes.GetEntityInfo(d.Context, tx, targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to fetch entity info", zap.Error(err), zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
		return resp.BadRequest("Failed to fetch entity info.")
	}

	// Now check the vote
	vi, err := votes.EntityVoteCheck(d.Context, tx, d.Auth.ID, targetId, targetType)

	if err != nil {
		return resp.Err("Failed to check vote", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	if !vi.VoteInfo.SupportsDownvotes && !upvote {
		return resp.BadRequest("This entity does not support downvotes")
	}

	if !vi.VoteInfo.SupportsUpvotes && upvote {
		return resp.BadRequest("This entity does not support upvotes")
	}

	if vi.HasVoted {
		// lacking MultipleVotes means that there can only be one vote per user for the entity
		if !vi.VoteInfo.MultipleVotes {
			if vi.ValidVotes[0].Upvote == upvote {
				return resp.BadRequest("You have already voted for this entity before!")
			} else {
				// Remove all old votes by said user
				_, err = tx.Exec(d.Context, "DELETE FROM entity_votes WHERE author = $1 AND target_id = $2 AND target_type = $3", d.Auth.ID, targetId, targetType)

				if err != nil {
					return resp.ErrDetail("Failed to delete old vote", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
				}
			}
		} else {
			var timeStr string
			if vi.Wait != nil {
				timeStr = fmt.Sprintf("%02d hours, %02d minutes. %02d seconds", vi.Wait.Hours, vi.Wait.Minutes, vi.Wait.Seconds)
			} else {
				timeStr = "a while"
			}

			if len(vi.ValidVotes) > 1 {
				return resp.BadRequest("Please wait " + timeStr + " before voting again. Your last vote was also a double vote!")
			}

			return resp.BadRequest("Please wait " + timeStr + " before voting again")
		}
	}

	// Keep adding votes until, but not including vi.VoteInfo.PerUser
	err = votes.EntityGiveVotes(d.Context, tx, upvote, d.Auth.ID, targetType, targetId, vi.VoteInfo)

	if err != nil {
		return resp.ErrDetail("Failed to give votes", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	// Perform post-vote tasks
	err = votes.EntityPostVote(d.Context, tx, targetType, targetId)

	if err != nil {
		return resp.ErrDetail("Failed to perform post-vote tasks", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	// Fetch new vote count
	nvc, err := votes.EntityGetVoteCount(d.Context, tx, targetId, targetType)

	if err != nil {
		return resp.ErrDetail("Failed to fetch new vote count", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	// Commit transaction
	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Failed to commit transaction", err, zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	// Fetch user info to log it to server
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while sending vote log message", zap.Any("panic", rec), zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
			}
		}()

		userObj, err := dovewing.GetUser(d.Context, d.Auth.ID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to fetch user info", zap.Error(err), zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
			return
		}

		_, err = state.Discord.Rest().CreateMessage(state.Config.Channels.VoteLogs, discord.MessageCreate{
			Embeds: []discord.Embed{
				{
					URL: entityInfo.URL,
					Thumbnail: &discord.EmbedResource{
						URL: entityInfo.Avatar,
					},
					Title:       "🎉 Vote Count Updated!",
					Description: ":heart:" + userObj.DisplayName + " has voted for " + targetType + ": " + entityInfo.Name,
					Color:       0x8A6BFD,
					Fields: []discord.EmbedField{
						{
							Name:   "Vote Count:",
							Value:  strconv.Itoa(nvc),
							Inline: validators.TruePtr,
						},
						{
							Name:   "Votes Added:",
							Value:  strconv.Itoa(vi.VoteInfo.PerUser),
							Inline: validators.TruePtr,
						},
						{
							Name:   "User ID:",
							Value:  userObj.ID,
							Inline: validators.TruePtr,
						},
						{
							Name:   "View " + targetType + "'s page",
							Value:  "[View " + entityInfo.Name + "](" + entityInfo.URL + ")",
							Inline: validators.TruePtr,
						},
					},
				},
			},
		})

		if err != nil {
			state.Logger.Error("Failed to send vote log message", zap.Error(err), zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
		}
	}()

	// Send webhook in a goroutine
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while sending vote webhook", zap.Any("panic", rec), zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
			}
		}()

		err := drivers.Send(drivers.With{
			UserID:     d.Auth.ID,
			TargetID:   targetId,
			TargetType: targetType,
			Data: events.WebhookNewVoteData{
				Votes:   nvc,
				PerUser: vi.VoteInfo.PerUser,
			},
		})

		if err != nil {
			state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("userId", d.Auth.ID), zap.String("targetId", targetId), zap.String("targetType", targetType))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
