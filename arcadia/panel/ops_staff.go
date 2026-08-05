package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"popplio/arcadia/dclient"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// correspondingServerNamesDebug reproduces Rust's `{:#?}` rendering of
// CorrespondingServer::VARIANTS, which is what the error message interpolates.
func correspondingServerNamesDebug() string {
	out := "[\n"

	for _, name := range types.CorrespondingServerNames {
		out += fmt.Sprintf("    %q,\n", name)
	}

	return out + "]"
}

func correspondingGuild(server types.CorrespondingServer) snowflake.ID {
	switch server {
	case types.CorrespondingServerMain:
		return state.Config.Servers.Main
	case types.CorrespondingServerTesting:
		return state.Config.Servers.Testing
	case types.CorrespondingServerStaff:
		return state.Config.Servers.Staff
	default:
		return 0
	}
}

// roleExists checks the Discord cache. An uncached guild reads as "role does not
// exist", as upstream does.
func roleExists(guildID, roleID snowflake.ID) bool {
	_, ok := dclient.Get().Caches().Role(guildID, roleID)
	return ok
}

// validateRoles checks the staff-server role and every corresponding role. It
// returns a response when validation fails.
func validateRoles(roleID string, correspondingRoles []types.Link) (*response, error) {
	roleSnow, err := snowflake.Parse(roleID)

	if err != nil {
		return nil, newError(err)
	}

	if !roleExists(state.Config.Servers.Staff, roleSnow) {
		resp := writeText(http.StatusBadRequest, "Role does not exist on the staff server")
		return &resp, nil
	}

	for _, role := range correspondingRoles {
		corrServer, err := types.CorrespondingServerFromString(role.Name)

		if err != nil {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf(
				"Server %s is not a supported corresponding role. Supported: %s",
				role.Name, correspondingServerNamesDebug(),
			))

			return &resp, nil
		}

		corrRoleSnow, err := snowflake.Parse(role.Value)

		if err != nil {
			return nil, newError(err)
		}

		guildID := correspondingGuild(corrServer)

		if !roleExists(guildID, corrRoleSnow) {
			resp := writeText(http.StatusBadRequest, fmt.Sprintf(
				"Role %s does not exist on the server %s", corrRoleSnow, guildID,
			))

			return &resp, nil
		}
	}

	return nil, nil
}

func (s *Server) updateStaffPositions(ctx context.Context, q *types.QUpdateStaffPositions) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	// ListPositions needs no permissions, so the (heavy) staff member lookup is
	// deferred until an action that actually needs it.
	if q.Action.ListPositions != nil {
		rows, err := state.Pool.Query(ctx,
			"SELECT id, name, role_id, perms, corresponding_roles, icon, index, created_at FROM staff_positions ORDER BY index ASC")

		if err != nil {
			return response{}, newError(fmt.Errorf("Error while getting staff positions %s", err))
		}

		positionRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[positionRow])

		if err != nil {
			return response{}, newError(fmt.Errorf("Error while getting staff positions %s", err))
		}

		positions := make([]types.StaffPosition, 0, len(positionRows))

		for _, p := range positionRows {
			var links []types.Link

			if err := json.Unmarshal(p.CorrespondingRoles, &links); err != nil {
				return response{}, newError(err)
			}

			positions = append(positions, types.StaffPosition{
				ID:                 impls.UUIDString(p.ID),
				Name:               p.Name,
				RoleID:             p.RoleID,
				Perms:              types.NonNilStrings(p.Perms),
				CorrespondingRoles: types.NonNilLinks(links),
				Icon:               p.Icon,
				Index:              p.Index,
				CreatedAt:          types.NewTimestamp(p.CreatedAt),
			})
		}

		return writeJSON(http.StatusOK, positions), nil
	}

	// These use the FULL resolution (including disciplinaries), not the light
	// path, so disciplinary permission limits apply here.
	sm, err := impls.GetStaffMember(ctx, authData.UserID)

	if err != nil {
		return response{}, newError(err)
	}

	smPerms := perms.Staff.SetFromStrings(sm.ResolvedPerms)
	// Rank comes from the member's grants rather than their positions, because
	// an instance owner outranks every position without holding one.
	smLowest := sm.Grants.Rank()

	switch {
	case q.Action.SwapIndex != nil:
		return s.swapIndex(ctx, q.Action.SwapIndex, smPerms, smLowest)
	case q.Action.SetIndex != nil:
		return s.setIndex(ctx, q.Action.SetIndex, smPerms, smLowest)
	case q.Action.CreatePosition != nil:
		return s.createPosition(ctx, q.Action.CreatePosition, smPerms, smLowest)
	case q.Action.EditPosition != nil:
		return s.editPosition(ctx, q.Action.EditPosition, smPerms, smLowest)
	case q.Action.DeletePosition != nil:
		return s.deletePosition(ctx, q.Action.DeletePosition, smPerms, smLowest)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No staff position action was specified")
	}
}

type positionRow struct {
	ID                 pgtype.UUID `db:"id"`
	Name               string      `db:"name"`
	RoleID             string      `db:"role_id"`
	Perms              []string    `db:"perms"`
	CorrespondingRoles []byte      `db:"corresponding_roles"`
	Icon               string      `db:"icon"`
	Index              int32       `db:"index"`
	CreatedAt          time.Time   `db:"created_at"`
}

func (s *Server) swapIndex(ctx context.Context, action *types.StaffSwapIndex, smPerms perms.Set, smLowest int32) (response, error) {
	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to swap indexes of staff positions [manage_staff_roles]"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	// Several handlers return early mid-transaction and rely on drop-rollback.
	defer tx.Rollback(ctx)

	var indexA int32

	if err := tx.QueryRow(ctx, "SELECT index FROM staff_positions WHERE id::text = $1", action.A).Scan(&indexA); err != nil {
		return response{}, newError(fmt.Errorf("Error while getting lower position %s", err))
	}

	var indexB int32

	if err := tx.QueryRow(ctx, "SELECT index FROM staff_positions WHERE id::text = $1", action.B).Scan(&indexB); err != nil {
		return response{}, newError(fmt.Errorf("Error while getting higher position %s", err))
	}

	if indexA == indexB {
		return writeText(http.StatusBadRequest, "Positions have the same index"), nil
	}

	if indexA <= smLowest || indexB <= smLowest {
		return writeText(http.StatusForbidden, "Either 'a' or 'b' is lower than the lowest index of the member"), nil
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_positions SET index = $1 WHERE id::text = $2", indexB, action.A); err != nil {
		return response{}, newError(fmt.Errorf("Error while updating lower position %s", err))
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_positions SET index = $1 WHERE id::text = $2", indexA, action.B); err != nil {
		return response{}, newError(fmt.Errorf("Error while updating higher position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) setIndex(ctx context.Context, action *types.StaffSetIndex, smPerms perms.Set, smLowest int32) (response, error) {
	id, err := uuid.Parse(action.ID)

	if err != nil {
		return response{}, newError(err)
	}

	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to set the indexes of staff positions [manage_staff_roles]"), nil
	}

	if action.Index < 0 {
		return writeText(http.StatusBadRequest, "Index cannot be lower than 0"), nil
	}

	if action.Index <= smLowest {
		return writeText(http.StatusForbidden, "Index to set is lower than or equal to the lowest index of the staff member"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var currIndex int32

	if err := tx.QueryRow(ctx, "SELECT index FROM staff_positions WHERE id = $1", id).Scan(&currIndex); err != nil {
		return response{}, newError(fmt.Errorf("Error while getting position %s", err))
	}

	if currIndex <= smLowest {
		return writeText(http.StatusForbidden, "Current index of position is lower than or equal to the lowest index of the staff member"), nil
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_positions SET index = index + 1 WHERE index >= $1", action.Index); err != nil {
		return response{}, newError(fmt.Errorf("Error while shifting indexes %s", err))
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_positions SET index = $1 WHERE id = $2", action.Index, id); err != nil {
		return response{}, newError(fmt.Errorf("Error while updating position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) createPosition(ctx context.Context, action *types.StaffCreatePosition, smPerms perms.Set, smLowest int32) (response, error) {
	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to create staff positions [manage_staff_roles]"), nil
	}

	if action.Index < 0 {
		return writeText(http.StatusBadRequest, "Index cannot be lower than 0"), nil
	}

	if action.Index <= smLowest {
		return writeText(http.StatusForbidden, "Index is lower than or equal to the lowest index of the staff member"), nil
	}

	if err := perms.Staff.ValidateStrings(action.Perms); err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	// The index shift below happens BEFORE role validation, so a validation
	// failure returns with the shift already executed. The deferred rollback is
	// what keeps that from being committed - upstream relies on the same
	// drop-rollback behaviour.
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE staff_positions SET index = index + 1 WHERE index >= $1", action.Index); err != nil {
		return response{}, newError(fmt.Errorf("Error while shifting indexes %s", err))
	}

	if resp, err := validateRoles(action.RoleID, action.CorrespondingRoles); err != nil {
		return response{}, err
	} else if resp != nil {
		return *resp, nil
	}

	correspondingRoles, err := json.Marshal(action.CorrespondingRoles)

	if err != nil {
		return response{}, newError(err)
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO staff_positions (name, perms, corresponding_roles, icon, role_id, index) VALUES ($1, $2, $3, $4, $5, $6)",
		action.Name, action.Perms, correspondingRoles, action.Icon, action.RoleID, action.Index)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while updating position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) editPosition(ctx context.Context, action *types.StaffEditPosition, smPerms perms.Set, smLowest int32) (response, error) {
	id, err := uuid.Parse(action.ID)

	if err != nil {
		return response{}, newError(err)
	}

	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to edit staff positions [manage_staff_roles]"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var (
		oldPerms []string
		index    int32
		roleID   string
	)

	err = tx.QueryRow(ctx, "SELECT perms, index, role_id FROM staff_positions WHERE id = $1 FOR UPDATE", id).Scan(&oldPerms, &index, &roleID)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting position %s", err))
	}

	if index <= smLowest {
		return writeText(http.StatusForbidden, "Index is lower than the lowest index of the member"), nil
	}

	if err := perms.Staff.ValidateStrings(action.Perms); err != nil {
		return writeText(http.StatusBadRequest, err.Error()), nil
	}

	if err := perms.CheckPatch(smPerms, perms.Staff.SetFromStrings(oldPerms), perms.Staff.SetFromStrings(action.Perms)); err != nil {
		return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms: %s", err)), nil
	}

	if resp, err := validateRoles(action.RoleID, action.CorrespondingRoles); err != nil {
		return response{}, err
	} else if resp != nil {
		return *resp, nil
	}

	correspondingRoles, err := json.Marshal(action.CorrespondingRoles)

	if err != nil {
		return response{}, newError(err)
	}

	_, err = tx.Exec(ctx,
		"UPDATE staff_positions SET name = $1, perms = $2, corresponding_roles = $3, role_id = $4, icon = $5 WHERE id = $6",
		action.Name, action.Perms, correspondingRoles, action.RoleID, action.Icon, id)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while updating position %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) deletePosition(ctx context.Context, action *types.StaffDeletePosition, smPerms perms.Set, smLowest int32) (response, error) {
	id, err := uuid.Parse(action.ID)

	if err != nil {
		return response{}, newError(err)
	}

	if !smPerms.Has(perms.StaffManageStaffRoles) {
		return writeText(http.StatusForbidden, "You do not have permission to delete staff positions [manage_staff_roles]"), nil
	}

	tx, err := state.Pool.Begin(ctx)

	if err != nil {
		return response{}, newError(err)
	}

	defer tx.Rollback(ctx)

	var (
		oldPerms []string
		index    int32
		roleID   string
	)

	err = tx.QueryRow(ctx, "SELECT perms, index, role_id FROM staff_positions WHERE id = $1 FOR UPDATE", id).Scan(&oldPerms, &index, &roleID)

	if err != nil {
		return response{}, newError(fmt.Errorf("Error while getting position %s", err))
	}

	if index <= smLowest {
		return writeText(http.StatusForbidden, "Index is lower than the lowest index of the member"), nil
	}

	// "neeed" is misspelled upstream and the panel shows the message raw.
	if err := perms.CheckPatch(smPerms, perms.Staff.SetFromStrings(oldPerms), perms.Staff.NewSet()); err != nil {
		return writeText(http.StatusForbidden, fmt.Sprintf("You do not have permission to edit the following perms [neeed to delete position]: %s", err)), nil
	}

	if _, err := tx.Exec(ctx, "DELETE FROM staff_positions WHERE id = $1", id); err != nil {
		return response{}, newError(fmt.Errorf("Error while deleting position %s", err))
	}

	if _, err := tx.Exec(ctx, "UPDATE staff_positions SET index = index - 1 WHERE index > $1", index); err != nil {
		return response{}, newError(fmt.Errorf("Error while shifting indexes %s", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

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
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

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

func countExists(ctx context.Context, query string, args ...any) (bool, error) {
	var count int64

	if err := state.Pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// requireRow is the "does this id exist" guard the shop, tier, disciplinary-type
// and whitelist mutations all run before updating or deleting. Every one of them
// reports the same frozen message, so the check lives in one place.
//
// It returns a response to send when the row is absent, nil when it is present.
func requireRow(ctx context.Context, query, id string) (*response, error) {
	exists, err := countExists(ctx, query, id)

	if err != nil {
		return nil, newError(err)
	}

	if !exists {
		resp := writeText(http.StatusBadRequest, "Entry with same id does not already exist")
		return &resp, nil
	}

	return nil, nil
}
