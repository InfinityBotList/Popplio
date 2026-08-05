// Package impls provides Arcadia's concrete implementations of the
// interfaces the panel depends on.
//
// Keeping authentication, permissions, entity lookup and cryptography behind
// interfaces is what lets the panel be exercised in tests without a live
// database or Discord connection; this package holds the production side of
// those seams.
package impls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/types"
	"popplio/perms"
	"popplio/state"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrIdentityExpired and ErrSessionNotActive are matched verbatim by the panel
// frontend, so the strings are frozen.
var (
	ErrIdentityExpired  = errors.New("identityExpired")
	ErrSessionNotActive = errors.New("sessionNotActive")
)

// CheckAuthInsecure validates a login token without requiring the session to be
// active.
//
// Steps 1 and 2 are the session garbage collector and they run on EVERY
// authenticated request, exactly as upstream does.
func CheckAuthInsecure(ctx context.Context, token string) (types.AuthData, error) {
	_, err := state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE created_at < NOW() - INTERVAL '1 hour'")

	if err != nil {
		return types.AuthData{}, err
	}

	_, err = state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE state = 'pending' AND created_at < NOW() - INTERVAL '5 minutes'")

	if err != nil {
		return types.AuthData{}, err
	}

	var count int64

	err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM staffpanel__authchain WHERE token = $1", token).Scan(&count)

	if err != nil {
		return types.AuthData{}, err
	}

	if count == 0 {
		return types.AuthData{}, ErrIdentityExpired
	}

	var (
		userID    string
		createdAt time.Time
		sessState string
	)

	err = state.Pool.QueryRow(ctx, "SELECT user_id, created_at, state FROM staffpanel__authchain WHERE token = $1", token).Scan(&userID, &createdAt, &sessState)

	if err != nil {
		return types.AuthData{}, err
	}

	var (
		positions []pgtype.UUID
		isBot     bool
	)

	// The bot flag rides along on this query rather than costing a round trip of
	// its own, since it is read on every authenticated request.
	err = state.Pool.QueryRow(ctx, `
		SELECT sm.positions, COALESCE(iuc.bot, false)
		FROM staff_members sm
		LEFT JOIN internal_user_cache__discord iuc ON iuc.id = sm.user_id
		WHERE sm.user_id = $1`, userID).Scan(&positions, &isBot)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.AuthData{}, ErrIdentityExpired
		}

		return types.AuthData{}, err
	}

	if len(positions) == 0 {
		return types.AuthData{}, ErrIdentityExpired
	}

	// A bot holds no staff permissions, so a session on one is not a session at
	// all — however it came to exist.
	if isBot {
		return types.AuthData{}, ErrIdentityExpired
	}

	return types.AuthData{
		UserID:    userID,
		CreatedAt: createdAt.Unix(),
		State:     sessState,
	}, nil
}

// CheckAuth is CheckAuthInsecure plus a requirement that the session is active.
func CheckAuth(ctx context.Context, token string) (types.AuthData, error) {
	data, err := CheckAuthInsecure(ctx, token)

	if err != nil {
		return types.AuthData{}, err
	}

	if data.State != "active" {
		return types.AuthData{}, ErrSessionNotActive
	}

	return data, nil
}

// disciplinaryRow is a disciplinary joined to its type. The type columns are
// nullable only because the join is a LEFT JOIN; a missing type is an error, as
// it is upstream.
type disciplinaryRow struct {
	ID          pgtype.UUID `db:"id"`
	CreatedAt   time.Time   `db:"created_at"`
	Expiry      *float64    `db:"expiry"`
	Title       string      `db:"title"`
	Description string      `db:"description"`
	Type        string      `db:"type"`

	TypeName           *string  `db:"type_name"`
	TypeDescription    *string  `db:"type_description"`
	TypeSelfAssignable *bool    `db:"self_assignable"`
	TypePermLimits     []string `db:"perm_limits"`
	TypeAdditory       *bool    `db:"additory"`
	TypeNeedsApproval  *bool    `db:"needs_approval"`
	TypeMaxExpiry      *float64 `db:"max_expiry"`
}

// disciplinaryQuery joins each disciplinary to its type in one round trip.
// Upstream issued a separate query per distinct type, memoized per call.
const disciplinaryQuery = `SELECT d.id, d.created_at, EXTRACT(epoch FROM d.expiry) AS expiry, d.title, d.description, d.type,
        t.name AS type_name, t.description AS type_description, t.self_assignable, t.perm_limits, t.additory, t.needs_approval,
        EXTRACT(epoch FROM t.max_expiry) AS max_expiry
        FROM staff_disciplinary d LEFT JOIN staff_disciplinary_types t ON t.id = d.type
        WHERE d.user_id = $1`

// GetStaffDisciplinaries loads a member's disciplinary actions with their types.
func GetStaffDisciplinaries(ctx context.Context, userID string, active bool) ([]types.StaffDisciplinary, error) {
	query := disciplinaryQuery

	if active {
		query += " AND NOW() - d.created_at < d.expiry"
	}

	rows, err := state.Pool.Query(ctx, query, userID)

	if err != nil {
		return nil, err
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[disciplinaryRow])

	if err != nil {
		return nil, err
	}

	disciplinaries := make([]types.StaffDisciplinary, 0, len(records))

	for _, rec := range records {
		if rec.TypeName == nil {
			// Upstream fetch_one's the type and propagates RowNotFound.
			return nil, pgx.ErrNoRows
		}

		var expiresAt *int64

		if rec.Expiry != nil {
			seconds := int64(*rec.Expiry)
			expiresAt = &seconds
		}

		disciplinaries = append(disciplinaries, types.StaffDisciplinary{
			ID:          UUIDString(rec.ID),
			UserID:      userID,
			CreatedAt:   types.NewTimestamp(rec.CreatedAt),
			ExpiresAt:   expiresAt,
			Title:       rec.Title,
			Description: rec.Description,
			Type: types.StaffDisciplinaryType{
				ID:             rec.Type,
				Name:           *rec.TypeName,
				Description:    derefOr(rec.TypeDescription, ""),
				SelfAssignable: derefOr(rec.TypeSelfAssignable, false),
				PermLimits:     types.NonNilStrings(rec.TypePermLimits),
				Additory:       derefOr(rec.TypeAdditory, false),
				NeedsApproval:  derefOr(rec.TypeNeedsApproval, false),
				MaxExpiry:      rec.TypeMaxExpiry,
				// QUIRK (reproduced): created_at is taken from the DISCIPLINARY, not
				// from the type row. See CONFORMANCE.md.
				CreatedAt: types.NewTimestamp(rec.CreatedAt),
			},
		})
	}

	return disciplinaries, nil
}

// derefOr reads through a nullable column, falling back when it is NULL.
func derefOr[T any](v *T, fallback T) T {
	if v == nil {
		return fallback
	}

	return *v
}

type staffPositionRow struct {
	ID                 pgtype.UUID  `db:"id"`
	Name               string       `db:"name"`
	RoleID             string       `db:"role_id"`
	Perms              []string     `db:"perms"`
	CorrespondingRoles []types.Link `db:"corresponding_roles"`
	Icon               string       `db:"icon"`
	Index              int32        `db:"index"`
	CreatedAt          time.Time    `db:"created_at"`
}

// GetStaffMember is the HEAVY permission path: positions, overrides AND active
// disciplinaries, plus the dovewing user.
func GetStaffMember(ctx context.Context, userID string) (types.StaffMember, error) {
	var (
		positionIDs   []pgtype.UUID
		permOverrides []string
		noAutosync    bool
		unaccounted   bool
		mfaVerified   bool
		createdAt     time.Time
	)

	err := state.Pool.QueryRow(ctx,
		"SELECT positions, perm_overrides, no_autosync, unaccounted, mfa_verified, created_at FROM staff_members WHERE user_id = $1",
		userID,
	).Scan(&positionIDs, &permOverrides, &noAutosync, &unaccounted, &mfaVerified, &createdAt)

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting staff perms of user %s: %s", userID, err)
	}

	rows, err := state.Pool.Query(ctx,
		"SELECT id, name, role_id, perms, corresponding_roles, icon, index, created_at FROM staff_positions WHERE id = ANY($1)",
		positionIDs,
	)

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting positions of user %s: %s", userID, err)
	}

	positionRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[staffPositionRow])

	if err != nil {
		return types.StaffMember{}, fmt.Errorf("Error while getting positions of user %s: %s", userID, err)
	}

	// The dovewing user is fetched before the permissions are resolved rather
	// than alongside the response, because whether this is a bot decides what
	// resolving can produce at all.
	user, err := GetPlatformUser(ctx, userID)

	if err != nil {
		return types.StaffMember{}, err
	}

	grants := perms.StaffGrants{
		Roles:       make([]perms.Role, 0, len(positionRows)),
		Extras:      perms.ParseStrings(permOverrides),
		ConfigOwner: perms.IsConfigOwner(userID),
		BotAccount:  user.Bot,
	}

	positions := make([]types.StaffPosition, 0, len(positionRows))

	for _, p := range positionRows {
		id := UUIDString(p.ID)

		grants.Roles = append(grants.Roles, perms.Role{
			ID:    id,
			Name:  p.Name,
			Index: p.Index,
			Perms: perms.ParseStrings(p.Perms),
		})

		positions = append(positions, types.StaffPosition{
			ID:                 id,
			Name:               p.Name,
			RoleID:             p.RoleID,
			Perms:              types.NonNilStrings(p.Perms),
			CorrespondingRoles: types.NonNilLinks(p.CorrespondingRoles),
			Icon:               p.Icon,
			Index:              p.Index,
			CreatedAt:          types.NewTimestamp(p.CreatedAt),
		})
	}

	disciplinaries, err := GetStaffDisciplinaries(ctx, userID, true)

	if err != nil {
		return types.StaffMember{}, err
	}

	resolved := resolveWithDisciplinaries(grants, disciplinaries)

	return types.StaffMember{
		UserID:         userID,
		User:           user,
		Positions:      positions,
		Disciplinaries: disciplinaries,
		PermOverrides:  types.NonNilStrings(permOverrides),
		ResolvedPerms:  resolved.Strings(),
		NoAutosync:     noAutosync,
		Unaccounted:    unaccounted,
		MfaVerified:    mfaVerified,
		CreatedAt:      types.NewTimestamp(createdAt),
		Grants:         grants,
	}, nil
}

// resolveWithDisciplinaries applies disciplinary permission limits on top of a
// member's own permissions.
//
// An additory disciplinary adds its permissions to what the member already has,
// which is how a disciplinary can hand someone a restricted extra ability. A
// non-additory one replaces their permissions instead: their roles and extras
// are set aside, leaving only what this disciplinary and any earlier one allow.
// That is what makes it a punishment — a suspended reviewer keeps nothing but
// the limits their disciplinary spells out.
func resolveWithDisciplinaries(grants perms.StaffGrants, disciplinaries []types.StaffDisciplinary) perms.Set {
	// A bot holds nothing, and an additory disciplinary adds permissions on top
	// of what is resolved — so a bot is answered before one can be applied.
	if grants.BotAccount {
		return perms.Staff.NewSet()
	}

	// An instance owner cannot be disciplined out of their own instance.
	if len(disciplinaries) == 0 || grants.ConfigOwner {
		return grants.Resolve()
	}

	var (
		resolved = grants.Resolve()
		// applied holds the disciplinary limits on their own, so that a
		// non-additory disciplinary can fall back to them.
		applied = perms.Staff.NewSet()
	)

	for _, disc := range disciplinaries {
		limits := perms.ParseStrings(disc.Type.PermLimits)

		if !disc.Type.Additory {
			resolved = applied
		}

		resolved = resolved.With(limits...)
		applied = applied.With(limits...)
	}

	return resolved
}
