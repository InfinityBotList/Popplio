package panel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// Shop items: what can be bought.

type shopItemRow struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Cents       float64   `db:"cents"`
	TargetTypes []string  `db:"target_types"`
	Benefits    []string  `db:"benefits"`
	CreatedAt   time.Time `db:"created_at"`
	LastUpdated time.Time `db:"last_updated"`
	CreatedBy   string    `db:"created_by"`
	UpdatedBy   string    `db:"updated_by"`
	Duration    int32     `db:"duration"`
	Description string    `db:"description"`
}

// validateShopItem applies the shared checks and confirms every benefit exists.
func validateShopItem(ctx context.Context, action *types.ShopItemUpsert) (*response, error) {
	if action.Cents < 0 {
		resp := writeText(http.StatusBadRequest, "Cents cannot be lower than 0")
		return &resp, nil
	}

	if action.Duration < 0 {
		resp := writeText(http.StatusBadRequest, "Duration cannot be lower than 0")
		return &resp, nil
	}

	for _, benefit := range action.Benefits {
		exists, err := countExists(ctx, "SELECT COUNT(*) FROM shop_item_benefits WHERE id = $1", benefit)

		if err != nil {
			return nil, err
		}

		if !exists {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf("Benefit %s does not exist", benefit))
			return &resp, nil
		}
	}

	return nil, nil
}

func (s *Server) updateShopItems(ctx context.Context, q *types.QUpdateShopItems) (response, error) {
	authData, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx,
			"SELECT id, name, cents, target_types, benefits, created_at, last_updated, created_by, updated_by, duration, description FROM shop_items ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		itemRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[shopItemRow])

		if err != nil {
			return response{}, newError(err)
		}

		items := make([]types.ShopItem, 0, len(itemRows))

		for _, i := range itemRows {
			items = append(items, types.ShopItem{
				ID:          i.ID,
				Name:        i.Name,
				Description: i.Description,
				Cents:       i.Cents,
				TargetTypes: types.NonNilStrings(i.TargetTypes),
				Benefits:    types.NonNilStrings(i.Benefits),
				Duration:    i.Duration,
				CreatedAt:   types.NewTimestamp(i.CreatedAt),
				LastUpdated: types.NewTimestamp(i.LastUpdated),
				CreatedBy:   i.CreatedBy,
				UpdatedBy:   i.UpdatedBy,
			})
		}

		return writeJSON(http.StatusOK, items), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create shop items [manage_shop]"), nil
		}

		resp, err := validateShopItem(ctx, action)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"INSERT INTO shop_items (id, name, cents, target_types, benefits, created_by, updated_by, duration, description) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8)",
			action.ID, action.Name, action.Cents, action.TargetTypes, action.Benefits, authData.UserID, action.Duration, action.Description)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update shop items [manage_shop]"), nil
		}

		resp, err := validateShopItem(ctx, action)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM shop_items WHERE id = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE shop_items SET name = $1, cents = $2, target_types = $3, benefits = $4, last_updated = NOW(), updated_by = $5, duration = $6, description = $7 WHERE id = $8",
			action.Name, action.Cents, action.TargetTypes, action.Benefits, authData.UserID, action.Duration, action.Description, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete shop items [manage_shop]"), nil
		}

		id := q.Action.Delete.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM shop_items WHERE id = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM shop_items WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No shop item action was specified")
	}
}
