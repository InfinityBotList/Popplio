package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// The review queue as the panel sees it, and the user lookup beside it.
//
// partialBots is the batched manager resolution: upstream resolved each bot's
// managers with two queries inside the loop, so an N-bot response cost 2N round
// trips. It is three queries total here, with the same JSON out.

func (s *Server) getUser(ctx context.Context, q *types.QGetUser) (response, error) {
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	user, err := impls.GetPlatformUser(ctx, q.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	return writeJSON(http.StatusOK, user), nil
}

type botQueueRow struct {
	BotID            string     `db:"bot_id"`
	ClientID         string     `db:"client_id"`
	LastClaimed      *time.Time `db:"last_claimed"`
	ClaimedBy        *string    `db:"claimed_by"`
	Type             string     `db:"type"`
	ApprovalNote     string     `db:"approval_note"`
	Short            string     `db:"short"`
	Invite           string     `db:"invite"`
	ApproximateVotes int32      `db:"approximate_votes"`
	Shards           int32      `db:"shards"`
	Library          string     `db:"library"`
	InviteClicks     int32      `db:"invite_clicks"`
	Clicks           int32      `db:"clicks"`
	Servers          int32      `db:"servers"`
}

func (s *Server) botQueue(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	// Public to all staff: no permission check.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	rows, err := state.Pool.Query(ctx,
		`SELECT bot_id, client_id, last_claimed, claimed_by, type, approval_note, short,
                invite, approximate_votes, shards, library, invite_clicks, clicks, servers
                FROM bots WHERE type = 'pending' OR type = 'claimed' ORDER BY created_at`)

	if err != nil {
		return response{}, newError(err)
	}

	queue, err := pgx.CollectRows(rows, pgx.RowToStructByName[botQueueRow])

	if err != nil {
		return response{}, newError(err)
	}

	return s.partialBots(ctx, queue)
}

// partialBots renders a bot list.
//
// Upstream resolved each bot's managers with two queries inside the loop, so an
// N-bot response cost 2N round trips. The managers are batched here into three
// queries total; the emitted JSON and the per-bot error messages are unchanged.
func (s *Server) partialBots(ctx context.Context, queue []botQueueRow) (response, error) {
	botIDs := make([]string, 0, len(queue))

	for _, bot := range queue {
		botIDs = append(botIDs, bot.BotID)
	}

	managers, err := impls.GetBotManagers(ctx, botIDs)

	if err != nil {
		return response{}, newError(err)
	}

	bots := make([]types.PartialEntity, 0, len(queue))

	for _, bot := range queue {
		// Dovewing is fronted by a Redis hot cache, so this stays per-bot.
		user, err := impls.GetPlatformUser(ctx, bot.BotID)

		if err != nil {
			return response{}, newError(err)
		}

		bots = append(bots, types.PartialEntity{Bot: &types.PartialBot{
			BotID:        bot.BotID,
			ClientID:     bot.ClientID,
			User:         user,
			ClaimedBy:    bot.ClaimedBy,
			LastClaimed:  types.TimestampPtr(bot.LastClaimed),
			ApprovalNote: bot.ApprovalNote,
			Short:        bot.Short,
			Type:         bot.Type,
			Votes:        bot.ApproximateVotes,
			Shards:       bot.Shards,
			Library:      bot.Library,
			InviteClicks: bot.InviteClicks,
			Clicks:       bot.Clicks,
			Servers:      bot.Servers,
			Mentionable:  managers[bot.BotID].Mentionables(),
			Invite:       bot.Invite,
		}})
	}

	return writeJSON(http.StatusOK, bots), nil
}
