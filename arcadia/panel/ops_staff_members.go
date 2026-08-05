package panel

import (
	"context"
	"fmt"
	"net/http"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
)

// Staff members: the people, their direct permission grants and their sync
// settings.
//
// Which positions someone holds is not editable here — that follows their
// Discord roles through the resync task. Only what is layered on top is.

func (s *Server) updateStaffMembers(ctx context.Context, q *types.QUpdateStaffMembers) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListMembers != nil:
		// No permission check. This is heavy: per member it hits staff_positions,
		// staff_disciplinary, staff_disciplinary_types and dovewing.
		rows, err := state.Pool.Query(ctx, "SELECT user_id FROM staff_members")

		if err != nil {
			return response{}, newError(fmt.Errorf("Error while getting staff members %s", err))
		}

		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])

		if err != nil {
			return response{}, newError(fmt.Errorf("Error while getting staff members %s", err))
		}

		members := make([]types.StaffMember, 0, len(ids))

		for _, id := range ids {
			member, err := impls.GetStaffMember(ctx, id)

			if err != nil {
				return response{}, newError(err)
			}

			members = append(members, member)
		}

		return writeJSON(http.StatusOK, members), nil
	case q.Action.EditMember != nil:
		return s.editMember(ctx, authData.UserID, q.Action.EditMember)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No staff member action was specified")
	}
}

func (s *Server) editMember(ctx context.Context, callerID string, action *types.StaffEditMember) (response, error) {
	sm, err := impls.GetStaffMember(ctx, callerID)

	if err != nil {
		return response{}, newError(err)
	}

	target, err := impls.GetStaffMember(ctx, action.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	smPerms := perms.Staff.SetFromStrings(sm.ResolvedPerms)

	if !smPerms.Has(perms.StaffManageStaffMembers) {
		return writeText(http.StatusForbidden, "You do not have permission to edit staff members [manage_staff_members]"), nil
	}

	if target.Grants.Rank() < sm.Grants.Rank() {
		return writeText(http.StatusForbidden, "Target has a lower index than the member"), nil
	}

	// Bots hold no staff permissions, so there is nothing here to edit. Writing
	// the overrides anyway would leave a row that reads as a grant and resolves
	// to nothing.
	if target.User.Bot {
		return writeText(http.StatusForbidden, perms.ErrBotAccount.Error()), nil
	}

	if err := perms.Staff.ValidateStrings(action.PermOverrides); err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	// The target's roles are kept; only their extra permissions change.
	newResolved := perms.StaffGrants{
		Roles:  target.Grants.Roles,
		Extras: perms.ParseStrings(action.PermOverrides),
	}.Resolve()

	if err := perms.CheckPatch(smPerms, perms.Staff.SetFromStrings(target.ResolvedPerms), newResolved); err != nil {
		return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var (
		currOverrides   []string
		currNoAutosync  bool
		currUnaccounted bool
	)

	err = tx.QueryRow(ctx, "SELECT perm_overrides, no_autosync, unaccounted FROM staff_members WHERE user_id = $1 FOR UPDATE", action.UserID).
		Scan(&currOverrides, &currNoAutosync, &currUnaccounted)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting member %s", err))
	}

	_, err = tx.Exec(ctx,
		"UPDATE staff_members SET perm_overrides = $1, no_autosync = $2, unaccounted = $3 WHERE user_id = $4",
		action.PermOverrides, action.NoAutosync, action.Unaccounted, action.UserID)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while updating member %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}
