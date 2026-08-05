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

// Shop item benefits: what buying an item actually gets you. Items reference
// benefits by id, so a benefit cannot be deleted while an item still names it.

type shopItemBenefitRow struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	TargetTypes []string  `db:"target_types"`
	CreatedAt   time.Time `db:"created_at"`
	CreatedBy   string    `db:"created_by"`
	LastUpdated time.Time `db:"last_updated"`
	UpdatedBy   string    `db:"updated_by"`
}

func (s *Server) updateShopItemBenefits(ctx context.Context, q *types.QUpdateShopItemBenefits) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx,
			"SELECT id, name, description, target_types, created_at, created_by, last_updated, updated_by FROM shop_item_benefits ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		benefitRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[shopItemBenefitRow])

		if err != nil {
			return response{}, newError(err)
		}

		benefits := make([]types.ShopItemBenefit, 0, len(benefitRows))

		for _, b := range benefitRows {
			benefits = append(benefits, types.ShopItemBenefit{
				ID:          b.ID,
				Name:        b.Name,
				Description: b.Description,
				CreatedAt:   types.NewTimestamp(b.CreatedAt),
				LastUpdated: types.NewTimestamp(b.LastUpdated),
				TargetTypes: types.NonNilStrings(b.TargetTypes),
				CreatedBy:   b.CreatedBy,
				UpdatedBy:   b.UpdatedBy,
			})
		}

		return writeJSON(http.StatusOK, benefits), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create shop item benefits [manage_shop]"), nil
		}

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO shop_item_benefits (id, name, description, target_types, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6)",
			action.ID, action.Name, action.Description, action.TargetTypes, authData.UserID, authData.UserID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update shop item benefits [manage_shop]"), nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM shop_item_benefits WHERE id = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE shop_item_benefits SET name = $1, description = $2, last_updated = NOW(), updated_by = $3, target_types = $4 WHERE id = $5",
			action.Name, action.Description, authData.UserID, action.TargetTypes, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete shop item benefits [manage_shop]"), nil
		}

		id := q.Action.Delete.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM shop_item_benefits WHERE id = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		// Referential guard: a benefit still in use by a shop item cannot go.
		inUse, err := countExists(ctx, "SELECT COUNT(*) FROM shop_items WHERE $1 = ANY(benefits)", id)

		if err != nil {
			return response{}, newError(err)
		}

		if inUse {
			return writeText(http.StatusBadRequest, "Cannot delete benefit as it is used by shop items"), nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM shop_item_benefits WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No shop item benefit action was specified")
	}
}
