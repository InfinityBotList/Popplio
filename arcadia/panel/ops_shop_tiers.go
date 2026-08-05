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

// Vote credit tiers: how many votes convert into how much credit.
//
// Tiers are ordered by an integer position that has to stay dense and unique,
// which is what dedupTierPositions exists for — every write here goes through it.

type voteCreditTierRow struct {
	ID         string    `db:"id"`
	TargetType string    `db:"target_type"`
	Position   int32     `db:"position"`
	Cents      float64   `db:"cents"`
	Votes      int32     `db:"votes"`
	CreatedAt  time.Time `db:"created_at"`
}

// validateTier applies the shared range and target-type checks.
func validateTier(action *types.VoteCreditTierUpsert) *response {
	if action.Cents < 0 {
		resp := writeText(http.StatusBadRequest, "Cents cannot be lower than 0")
		return &resp
	}

	if action.Votes < 0 {
		resp := writeText(http.StatusBadRequest, "Votes cannot be lower than 0")
		return &resp
	}

	if action.TargetType != "bot" && action.TargetType != "server" {
		resp := writeText(http.StatusBadRequest, "Target type must be either 'bot' or 'server'")
		return &resp
	}

	return nil
}

// dedupTierPositions pushes conflicting tiers down until the given position is
// unique. This loop is load-bearing for ordering and is reproduced exactly.
func dedupTierPositions(ctx context.Context, tx pgx.Tx, position int32, id string) error {
	indexA := position

	for {
		rows, err := tx.Query(ctx, "SELECT id, position FROM vote_credit_tiers WHERE position = $1 AND id != $2", indexA, id)

		if err != nil {
			return err
		}

		type conflict struct {
			ID       string `db:"id"`
			Position int32  `db:"position"`
		}

		conflicts, err := pgx.CollectRows(rows, pgx.RowToStructByName[conflict])

		if err != nil {
			return err
		}

		if len(conflicts) == 0 {
			return nil
		}

		indexB := indexA + 1

		for _, c := range conflicts {
			if _, err := tx.Exec(ctx, "UPDATE vote_credit_tiers SET position = $1 WHERE id = $2", indexB, c.ID); err != nil {
				return err
			}

			indexB++
		}

		indexA = indexB
	}
}

func (s *Server) updateVoteCreditTiers(ctx context.Context, q *types.QUpdateVoteCreditTiers) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListTiers != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx, "SELECT id, target_type, position, cents, votes, created_at FROM vote_credit_tiers ORDER BY position ASC")

		if err != nil {
			return response{}, newError(err)
		}

		tierRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[voteCreditTierRow])

		if err != nil {
			return response{}, newError(err)
		}

		tiers := make([]types.VoteCreditTier, 0, len(tierRows))

		for _, t := range tierRows {
			tiers = append(tiers, types.VoteCreditTier{
				ID:         t.ID,
				TargetType: t.TargetType,
				Position:   t.Position,
				Cents:      t.Cents,
				Votes:      t.Votes,
				CreatedAt:  types.NewTimestamp(t.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, tiers), nil
	case q.Action.CreateTier != nil:
		action := q.Action.CreateTier

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create vote credit tiers [manage_shop]"), nil
		}

		if resp := validateTier(action); resp != nil {
			return *resp, nil
		}

		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx,
			"INSERT INTO vote_credit_tiers (id, target_type, position, cents, votes) VALUES ($1, $2, $3, $4, $5)",
			action.ID, action.TargetType, action.Position, action.Cents, action.Votes)

		if err != nil {
			return response{}, newError(err)
		}

		if err := dedupTierPositions(ctx, tx, action.Position, action.ID); err != nil {
			return response{}, newError(err)
		}

		if err := tx.Commit(ctx); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.EditTier != nil:
		action := q.Action.EditTier

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update vote credit tiers [manage_shop]"), nil
		}

		// Existence is checked BEFORE the range validations here.
		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM vote_credit_tiers WHERE id = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if resp := validateTier(action); resp != nil {
			return *resp, nil
		}

		tx, err := state.Pool.Begin(ctx)

		if err != nil {
			return response{}, newError(err)
		}

		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx,
			"UPDATE vote_credit_tiers SET position = $1, target_type = $2, cents = $3, votes = $4 WHERE id = $5",
			action.Position, action.TargetType, action.Cents, action.Votes, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		if err := dedupTierPositions(ctx, tx, action.Position, action.ID); err != nil {
			return response{}, newError(err)
		}

		if err := tx.Commit(ctx); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.DeleteTier != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete vote credit tiers [manage_shop]"), nil
		}

		id := q.Action.DeleteTier.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM vote_credit_tiers WHERE id = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM vote_credit_tiers WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No vote credit tier action was specified")
	}
}
