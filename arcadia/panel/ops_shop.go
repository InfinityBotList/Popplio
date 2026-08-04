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
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

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
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

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
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

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

type shopCouponRow struct {
	ID                string    `db:"id"`
	Code              string    `db:"code"`
	Public            bool      `db:"public"`
	MaxUses           *int32    `db:"max_uses"`
	CreatedAt         time.Time `db:"created_at"`
	CreatedBy         string    `db:"created_by"`
	LastUpdated       time.Time `db:"last_updated"`
	UpdatedBy         string    `db:"updated_by"`
	ReuseWaitDuration *int32    `db:"reuse_wait_duration"`
	Expiry            *int32    `db:"expiry"`
	ApplicableItems   []string  `db:"applicable_items"`
	Cents             *float64  `db:"cents"`
	Requirements      []string  `db:"requirements"`
	AllowedUsers      []string  `db:"allowed_users"`
	Usable            bool      `db:"usable"`
	TargetTypes       []string  `db:"target_types"`
}

// validateCoupon applies the shared coupon checks.
//
// BUG (reproduced): these use unwrap_or(0), so a NULL max_uses / expiry /
// reuse_wait_duration becomes 0 and FAILS its check. The "unlimited uses" and
// "never expires" cases the DTO comments describe are therefore unreachable
// through this endpoint. See CONFORMANCE.md, which also carries the one-line
// fix as an opt-in patch.
func validateCoupon(action *types.ShopCouponUpsert) *response {
	if derefOr(action.MaxUses, 0) <= 0 {
		resp := writeText(http.StatusBadRequest, "Max uses must be greater than 0")
		return &resp
	}

	if derefOr(action.ReuseWaitDuration, 0) <= 0 {
		resp := writeText(http.StatusBadRequest, "Reuse wait duration must be greater than 0")
		return &resp
	}

	if derefOr(action.Expiry, 0) <= 0 {
		resp := writeText(http.StatusBadRequest, "Expiry must be greater than 0")
		return &resp
	}

	if derefOr(action.Cents, 0.0) < 0 {
		resp := writeText(http.StatusBadRequest, "Cents cannot be lower than 0")
		return &resp
	}

	return nil
}

// derefOr reads through a nullable field, falling back when it is null. The
// coupon validations rely on the fallback being applied - see CONFORMANCE.md a/2.
func derefOr[T any](v *T, fallback T) T {
	if v == nil {
		return fallback
	}

	return *v
}

func validateCouponItems(ctx context.Context, items []string) (*response, error) {
	for _, item := range items {
		exists, err := countExists(ctx, "SELECT COUNT(*) FROM shop_items WHERE id = $1", item)

		if err != nil {
			return nil, err
		}

		if !exists {
			// Rust's `{:#?}` of a String renders with surrounding quotes.
			resp := writeText(http.StatusBadRequest, fmt.Sprintf("Item %q does not exist", item))
			return &resp, nil
		}
	}

	return nil, nil
}

func (s *Server) updateShopCoupons(ctx context.Context, q *types.QUpdateShopCoupons) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.List != nil:
		// Unlike the other List actions, this one REQUIRES a permission.
		if !userPerms.Has(perms.StaffViewShop) {
			return writeText(http.StatusForbidden, "You do not have permission to list shop coupons [view_shop]"), nil
		}

		rows, err := state.Pool.Query(ctx,
			"SELECT id, code, public, max_uses, created_at, created_by, last_updated, updated_by, reuse_wait_duration, expiry, applicable_items, cents, requirements, allowed_users, usable, target_types FROM shop_coupons ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		couponRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[shopCouponRow])

		if err != nil {
			return response{}, newError(err)
		}

		coupons := make([]types.ShopCoupon, 0, len(couponRows))

		for _, c := range couponRows {
			coupons = append(coupons, types.ShopCoupon{
				ID:                c.ID,
				Code:              c.Code,
				Public:            c.Public,
				MaxUses:           c.MaxUses,
				CreatedAt:         types.NewTimestamp(c.CreatedAt),
				CreatedBy:         c.CreatedBy,
				LastUpdated:       types.NewTimestamp(c.LastUpdated),
				UpdatedBy:         c.UpdatedBy,
				ReuseWaitDuration: c.ReuseWaitDuration,
				Expiry:            c.Expiry,
				ApplicableItems:   types.NonNilStrings(c.ApplicableItems),
				Cents:             c.Cents,
				Requirements:      types.NonNilStrings(c.Requirements),
				AllowedUsers:      types.NonNilStrings(c.AllowedUsers),
				Usable:            c.Usable,
				TargetTypes:       types.NonNilStrings(c.TargetTypes),
			})
		}

		return writeJSON(http.StatusOK, coupons), nil
	case q.Action.Create != nil:
		action := q.Action.Create

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to create shop coupons [manage_shop]"), nil
		}

		if resp := validateCoupon(action); resp != nil {
			return *resp, nil
		}

		resp, err := validateCouponItems(ctx, action.ApplicableItems)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"INSERT INTO shop_coupons (id, code, public, max_uses, created_by, updated_by, reuse_wait_duration, expiry, applicable_items, cents, requirements, allowed_users, usable, target_types) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9, $10, $11, $12, $13)",
			action.ID, action.Code, action.Public, action.MaxUses, authData.UserID,
			action.ReuseWaitDuration, action.Expiry, action.ApplicableItems, action.Cents,
			action.Requirements, action.AllowedUsers, action.Usable, action.TargetTypes)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Edit != nil:
		action := q.Action.Edit

		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to update shop coupons [manage_shop]"), nil
		}

		if resp := validateCoupon(action); resp != nil {
			return *resp, nil
		}

		resp, err := validateCouponItems(ctx, action.ApplicableItems)

		if err != nil {
			return response{}, newError(err)
		}

		if resp != nil {
			return *resp, nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM shop_coupons WHERE id = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE shop_coupons SET code = $1, public = $2, max_uses = $3, reuse_wait_duration = $4, expiry = $5, applicable_items = $6, cents = $7, requirements = $8, updated_by = $9, last_updated = NOW(), allowed_users = $10, usable = $11, target_types = $12 WHERE id = $13",
			action.Code, action.Public, action.MaxUses, action.ReuseWaitDuration, action.Expiry,
			action.ApplicableItems, action.Cents, action.Requirements, authData.UserID,
			action.AllowedUsers, action.Usable, action.TargetTypes, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.Delete != nil:
		if !userPerms.Has(perms.StaffManageShop) {
			return writeText(http.StatusForbidden, "You do not have permission to delete shop coupons [manage_shop]"), nil
		}

		id := q.Action.Delete.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM shop_coupons WHERE id = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM shop_coupons WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No shop coupon action was specified")
	}
}

type botWhitelistRow struct {
	BotID     string    `db:"bot_id"`
	UserID    string    `db:"user_id"`
	Reason    string    `db:"reason"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Server) updateBotWhitelist(ctx context.Context, q *types.QUpdateBotWhitelist) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

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
