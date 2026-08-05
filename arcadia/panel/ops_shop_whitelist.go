package panel

import (
	"context"
	"net/http"
	"time"

	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// The bot whitelist: bots allowed past a restriction they would otherwise hit.
//
// It lives with the shop operations because it is administered from the same
// panel section, not because it has anything to do with buying things.

type botWhitelistRow struct {
	BotID     string    `db:"bot_id"`
	UserID    string    `db:"user_id"`
	Reason    string    `db:"reason"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Server) updateBotWhitelist(ctx context.Context, q *types.QUpdateBotWhitelist) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx, "SELECT bot_id, user_id, reason, created_at FROM bot_whitelist ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		whitelistRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[botWhitelistRow])

		if err != nil {
			return response{}, newError(err)
		}

		entries := make([]types.BotWhitelist, 0, len(whitelistRows))

		for _, w := range whitelistRows {
			entries = append(entries, types.BotWhitelist{
				BotID:     w.BotID,
				UserID:    w.UserID,
				Reason:    w.Reason,
				CreatedAt: types.NewTimestamp(w.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, entries), nil
	case q.Action.Add != nil:
		action := q.Action.Add

		// Note the PARENTHESES here rather than the square brackets the other
		// messages use.
		if !userPerms.Has(perms.StaffManageBotWhitelist) {
			return writeText(http.StatusForbidden, "You do not have permission to add to the bot whitelist (bot_whitelist.create)"), nil
		}

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO bot_whitelist (user_id, bot_id, reason) VALUES ($1, $2, $3)",
			authData.UserID, action.BotID, action.Reason)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageBotWhitelist) {
			return writeText(http.StatusForbidden, "You do not have permission to update bot whitelist (bot_whitelist.update)"), nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM bot_whitelist WHERE bot_id = $1", action.BotID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "UPDATE bot_whitelist SET reason = $1 WHERE bot_id = $2", action.Reason, action.BotID); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageBotWhitelist) {
			return writeText(http.StatusForbidden, "You do not have permission to delete bot whitelist entries (bot_whitelist.delete)"), nil
		}

		botID := q.Action.Delete.BotID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM bot_whitelist WHERE bot_id = $1", botID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM bot_whitelist WHERE bot_id = $1", botID); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No bot whitelist action was specified")
	}
}
