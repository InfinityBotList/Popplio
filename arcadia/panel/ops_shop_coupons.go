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

// Shop coupons: discounts, optionally restricted to particular items and
// redeemable a limited number of times.

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
	authData, userPerms, err := authorize(ctx, q.LoginToken)

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
