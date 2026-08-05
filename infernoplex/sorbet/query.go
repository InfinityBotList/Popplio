package sorbet

import (
	"context"
	"fmt"
	"net/http"

	"popplio/infernoplex/dclient"
	"popplio/infernoplex/invite"

	"github.com/disgoorg/snowflake/v2"
)

type InfernoplexQuery struct {
	CreateInvite *QueryCreateInvite `json:"CreateInvite,omitempty"`

	ResolveInvite *QueryResolveInvite `json:"ResolveInvite,omitempty"`

	IsInGuild *QueryIsInGuild `json:"IsInGuild,omitempty"`
}

type QueryCreateInvite struct {
	Session *string `json:"session"`
	GuildID string  `json:"guild_id"`
}

type QueryResolveInvite struct {
	InviteCode string `json:"invite_code"`
	GuildID    string `json:"guild_id"`
}

type QueryIsInGuild struct {
	GuildID string `json:"guild_id"`
}

type result struct {
	status  int
	body    any
	headers map[string]string
}

func dispatch(ctx context.Context, req InfernoplexQuery) result {
	switch {
	case req.CreateInvite != nil:
		return dispatchCreateInvite(ctx, req.CreateInvite)
	case req.ResolveInvite != nil:
		return dispatchResolveInvite(ctx, req.ResolveInvite)
	case req.IsInGuild != nil:
		return dispatchIsInGuild(req.IsInGuild)
	default:
		return result{status: http.StatusBadRequest, body: map[string]string{"message": "no query variant was set"}}
	}
}

func dispatchCreateInvite(ctx context.Context, q *QueryCreateInvite) result {
	var userID *string

	if q.Session != nil {
		sess, err := sessionFromToken(ctx, *q.Session)

		if err != nil {
			return createInviteErrResult(http.StatusBadRequest, nil, &invite.CreateInviteError{
				Kind: invite.ErrGeneric, Message: fmt.Sprintf("Invalid session: %v", err),
			})
		}

		if sess == nil {
			return createInviteErrResult(http.StatusForbidden, map[string]string{"X-Session-Invalid": "1"}, &invite.CreateInviteError{
				Kind: invite.ErrGeneric, Message: "Invalid session token",
			})
		}

		if sess.TargetType != "user" {
			return createInviteErrResult(http.StatusForbidden, nil, &invite.CreateInviteError{
				Kind: invite.ErrGeneric, Message: "CreateInvite can only be called on a user session",
			})
		}

		userID = &sess.TargetID
	}

	guildID, err := snowflake.Parse(q.GuildID)

	if err != nil {
		return createInviteErrResult(http.StatusBadRequest, nil, &invite.CreateInviteError{
			Kind: invite.ErrGeneric, Message: fmt.Sprintf("Invalid guild ID: %v", err),
		})
	}

	url, cErr := invite.CreateInviteForUser(ctx, guildID, userID, false)

	if cErr != nil {
		return createInviteErrResult(http.StatusInternalServerError, nil, cErr)
	}

	return result{status: http.StatusOK, body: map[string]any{
		"CreateInvite": map[string]any{
			"result": map[string]any{
				"Invite": map[string]any{"url": url},
			},
		},
	}}
}

func createInviteErrResult(status int, headers map[string]string, err *invite.CreateInviteError) result {
	return result{
		status:  status,
		headers: headers,
		body: map[string]any{
			"CreateInvite": map[string]any{
				"err":     map[string]any{string(err.Kind): map[string]any{}},
				"message": err.Error(),
			},
		},
	}
}

func dispatchResolveInvite(ctx context.Context, q *QueryResolveInvite) result {
	guildID, err := snowflake.Parse(q.GuildID)

	if err != nil {
		return result{status: http.StatusBadRequest, body: map[string]any{
			"ResolveInvite": map[string]any{"message": fmt.Sprintf("Invalid guild ID: %v", err)},
		}}
	}

	if err := invite.ResolveInvite(ctx, guildID, q.InviteCode); err != nil {
		return result{status: http.StatusInternalServerError, body: map[string]any{
			"ResolveInvite": map[string]any{"message": err.Error()},
		}}
	}

	return result{status: http.StatusOK, body: map[string]any{"ResolveInvite": map[string]any{}}}
}

func dispatchIsInGuild(q *QueryIsInGuild) result {
	guildID, err := snowflake.Parse(q.GuildID)

	if err != nil {
		return result{status: http.StatusBadRequest, body: map[string]string{"message": fmt.Sprintf("Invalid guild ID: %v", err)}}
	}

	_, inGuild := dclient.Get().Caches().Guild(guildID)

	inviteURL := fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&scope=bot%%20applications.commands&guild_id=%s&disable_guild_select=true",
		dclient.Get().ApplicationID(), guildID,
	)

	return result{status: http.StatusOK, body: map[string]any{
		"IsInGuild": map[string]any{
			"in_guild":   inGuild,
			"invite_url": inviteURL,
		},
	}}
}
