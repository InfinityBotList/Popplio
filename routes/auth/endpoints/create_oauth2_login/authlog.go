package create_oauth2_login

import (
	"time"

	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/disgoorg/disgo/discord"
	"go.uber.org/zap"
)

// yesNo renders a boolean the way the auth-log embed displays it.
func yesNo(b bool) string {
	if b {
		return "Yes"
	}

	return "No"
}

// banState is a user's three independent ban flags, as shown in the auth log.
type banState struct {
	banned     bool
	voteBanned bool
	appBanned  bool
}

// sendAuthLog posts a login attempt to the staff auth-log channel.
//
// It is called for its side effect only and never fails the login: a login that
// succeeded must not be reported as failed just because the audit message could
// not be delivered. Failures are logged instead.
//
// isNew says whether the users row was created by this very login, in which case
// there are no ban flags to look up yet.
func sendAuthLog(user oauthUser, req types.AuthorizeRequest, isNew bool) {
	var bans banState

	if !isNew {
		err := state.Pool.QueryRow(
			state.Context,
			"SELECT banned, vote_banned, app_banned FROM users WHERE user_id = $1",
			user.ID,
		).Scan(&bans.banned, &bans.voteBanned, &bans.appBanned)

		if err != nil {
			state.Logger.Error("sendAuthLog: Failed to get user details from database", zap.Error(err), zap.String("user_id", user.ID))
			return
		}
	}

	state.Logger.With(
		zap.String("user_id", user.ID),
		zap.String("channel_id", state.Config.Channels.AuthLogs.String()),
	).Debug("sendAuthLog: Channel Info")

	_, err := state.Discord.Rest().CreateMessage(state.Config.Channels.AuthLogs, discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				Title:  "User Login Attempt",
				Color:  0xff0000,
				Fields: authLogFields(user, req, bans, isNew),
				Footer: &discord.EmbedFooter{
					Text: "© Copyright 2023 - Infinity Bot List",
				},
				Timestamp: validators.Pointer(time.Now()),
			},
		},
	})

	if err != nil {
		state.Logger.Error("sendAuthLog: Failed to send message to Discord", zap.Error(err), zap.String("user_id", user.ID))
	}
}

// authLogFields builds the embed fields describing a login attempt.
func authLogFields(user oauthUser, req types.AuthorizeRequest, bans banState, isNew bool) []discord.EmbedField {
	fields := []struct {
		name  string
		value string
	}{
		{"User ID", user.ID},
		{"Username", user.Username + "#" + user.Disc},
		{"Scope", req.Scope},
		{"Banned", yesNo(bans.banned)},
		{"Vote Banned", yesNo(bans.voteBanned)},
		{"App Banned", yesNo(bans.appBanned)},
		{"New User", yesNo(isNew)},
	}

	embedFields := make([]discord.EmbedField, 0, len(fields))
	for _, f := range fields {
		embedFields = append(embedFields, discord.EmbedField{
			Name:   f.name,
			Value:  f.value,
			Inline: validators.TruePtr,
		})
	}

	return embedFields
}
