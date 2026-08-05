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

// Disciplinary types: the punishments that can be issued, and the permission
// limits each one imposes while it is active.
//
// An additory type adds its limits to what the member already has; a
// non-additory one replaces them, which is what makes it a punishment. See
// resolveWithDisciplinaries in arcadia/impls/auth.go.

type disciplinaryTypeRow struct {
	ID             string    `db:"id"`
	Name           string    `db:"name"`
	Description    string    `db:"description"`
	SelfAssignable bool      `db:"self_assignable"`
	PermLimits     []string  `db:"perm_limits"`
	Additory       bool      `db:"additory"`
	NeedsApproval  bool      `db:"needs_approval"`
	MaxExpiry      *float64  `db:"max_expiry"`
	CreatedAt      time.Time `db:"created_at"`
}

func (s *Server) updateStaffDisciplinaryType(ctx context.Context, q *types.QUpdateStaffDisciplinaryType) (response, error) {
	_, userPerms, err := authorize(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	switch {
	case q.Action.ListDisciplinaryTypes != nil:
		// No permission check.
		rows, err := state.Pool.Query(ctx,
			"SELECT id, name, description, self_assignable, perm_limits, additory, needs_approval, EXTRACT(epoch FROM max_expiry) AS max_expiry, created_at FROM staff_disciplinary_types ORDER BY created_at DESC")

		if err != nil {
			return response{}, newError(err)
		}

		typeRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[disciplinaryTypeRow])

		if err != nil {
			return response{}, newError(err)
		}

		out := make([]types.StaffDisciplinaryType, 0, len(typeRows))

		for _, t := range typeRows {
			out = append(out, types.StaffDisciplinaryType{
				ID:             t.ID,
				Name:           t.Name,
				Description:    t.Description,
				SelfAssignable: t.SelfAssignable,
				PermLimits:     types.NonNilStrings(t.PermLimits),
				Additory:       t.Additory,
				NeedsApproval:  t.NeedsApproval,
				MaxExpiry:      t.MaxExpiry,
				CreatedAt:      types.NewTimestamp(t.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, out), nil
	case q.Action.CreateDisciplinaryType != nil:
		action := q.Action.CreateDisciplinaryType

		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to create staff disciplinary types [manage_disciplinaries]"), nil
		}

		if err := perms.Staff.ValidateStrings(action.PermLimits); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		if err := perms.CheckPatch(userPerms, perms.Staff.NewSet(), perms.Staff.SetFromStrings(action.PermLimits)); err != nil {
			return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
		}

		_, err := state.Pool.Exec(ctx,
			"INSERT INTO staff_disciplinary_types (id, name, description, self_assignable, perm_limits, additory, needs_approval, max_expiry) VALUES ($1, $2, $3, $4, $5, $6, $7, make_interval(secs => $8))",
			action.ID, action.Name, action.Description, action.SelfAssignable, action.PermLimits, action.Additory, action.NeedsApproval, action.MaxExpiry)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.EditDisciplinaryType != nil:
		action := q.Action.EditDisciplinaryType

		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to update staff disciplinary types [manage_disciplinaries]"), nil
		}

		if err := perms.Staff.ValidateStrings(action.PermLimits); err != nil {
			return writeText(http.StatusBadRequest, err.Error()), nil
		}

		if err := perms.CheckPatch(userPerms, perms.Staff.NewSet(), perms.Staff.SetFromStrings(action.PermLimits)); err != nil {
			return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
		}

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM staff_disciplinary_types WHERE id = $1", action.ID); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		_, err = state.Pool.Exec(ctx,
			"UPDATE staff_disciplinary_types SET name = $1, description = $2, self_assignable = $3, perm_limits = $4, additory = $5, needs_approval = $6, max_expiry = make_interval(secs => $7) WHERE id = $8",
			action.Name, action.Description, action.SelfAssignable, action.PermLimits, action.Additory, action.NeedsApproval, action.MaxExpiry, action.ID)

		if err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	case q.Action.DeleteDisciplinaryType != nil:
		if !userPerms.Has(perms.StaffManageDisciplinaries) {
			return writeText(http.StatusForbidden, "You do not have permission to delete staff disciplinary types [manage_disciplinaries]"), nil
		}

		id := q.Action.DeleteDisciplinaryType.ID

		if resp, err := requireRow(ctx, "SELECT COUNT(*) FROM staff_disciplinary_types WHERE id = $1", id); err != nil {
			return response{}, err
		} else if resp != nil {
			return *resp, nil
		}

		if _, err := state.Pool.Exec(ctx, "DELETE FROM staff_disciplinary_types WHERE id = $1", id); err != nil {
			return response{}, newError(err)
		}

		return writeNoContent(), nil
	default:
		return response{}, errStatus(http.StatusBadRequest, "No disciplinary type action was specified")
	}
}
