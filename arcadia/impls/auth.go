package impls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/arcadia/types"
	"popplio/state"

	perms "github.com/infinitybotlist/kittycat/go"
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

	var positions []pgtype.UUID

	err = state.Pool.QueryRow(ctx, "SELECT positions FROM staff_members WHERE user_id = $1", userID).Scan(&positions)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return types.AuthData{}, ErrIdentityExpired
		}

		return types.AuthData{}, err
	}

	if len(positions) == 0 {
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

type disciplinaryRow struct {
	ID          pgtype.UUID `db:"id"`
	CreatedAt   time.Time   `db:"created_at"`
	Expiry      *float64    `db:"expiry"`
	Title       string      `db:"title"`
	Description string      `db:"description"`
	Type        string      `db:"type"`
}

// GetStaffDisciplinaries loads a member's disciplinary actions, resolving each
// one's type (memoized per call, as upstream does).
func GetStaffDisciplinaries(ctx context.Context, userID string, active bool) ([]types.StaffDisciplinary, error) {
	query := "SELECT id, created_at, EXTRACT(epoch FROM expiry) as expiry, title, description, type FROM staff_disciplinary WHERE user_id = $1"

	if active {
		query += " AND NOW() - created_at < expiry"
	}

	rows, err := state.Pool.Query(ctx, query, userID)

	if err != nil {
		return nil, err
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[disciplinaryRow])

	if err != nil {
		return nil, err
	}

	typeCache := make(map[string]types.StaffDisciplinaryType)
	disciplinaries := make([]types.StaffDisciplinary, 0, len(records))

	for _, rec := range records {
		discType, ok := typeCache[rec.Type]

		if !ok {
			var (
				name           string
				description    string
				selfAssignable bool
				permLimits     []string
				additory       bool
				needsApproval  bool
				maxExpiry      *float64
			)

			err := state.Pool.QueryRow(ctx,
				"SELECT name, description, self_assignable, perm_limits, additory, needs_approval,  EXTRACT(epoch FROM max_expiry) AS max_expiry FROM staff_disciplinary_types WHERE id = $1",
				rec.Type,
			).Scan(&name, &description, &selfAssignable, &permLimits, &additory, &needsApproval, &maxExpiry)

			if err != nil {
				return nil, err
			}

			discType = types.StaffDisciplinaryType{
				ID:             rec.Type,
				Name:           name,
				Description:    description,
				SelfAssignable: selfAssignable,
				PermLimits:     nonNil(permLimits),
				Additory:       additory,
				NeedsApproval:  needsApproval,
				MaxExpiry:      maxExpiry,
				// QUIRK (reproduced): created_at is taken from the DISCIPLINARY, not
				// from the type row. See CONFORMANCE.md.
				CreatedAt: types.NewTimestamp(rec.CreatedAt),
			}

			typeCache[rec.Type] = discType
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
			Type:        discType,
		})
	}

	return disciplinaries, nil
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

	sp := perms.StaffPermissions{
		UserPositions: make([]perms.PartialStaffPosition, 0, len(positionRows)),
		PermOverrides: perms.PFSS(permOverrides),
	}

	positions := make([]types.StaffPosition, 0, len(positionRows))

	for _, p := range positionRows {
		id := UUIDString(p.ID)

		sp.UserPositions = append(sp.UserPositions, perms.PartialStaffPosition{
			ID:    id,
			Index: p.Index,
			Perms: perms.PFSS(p.Perms),
		})

		positions = append(positions, types.StaffPosition{
			ID:                 id,
			Name:               p.Name,
			RoleID:             p.RoleID,
			Perms:              nonNil(p.Perms),
			CorrespondingRoles: nonNilLinks(p.CorrespondingRoles),
			Icon:               p.Icon,
			Index:              p.Index,
			CreatedAt:          types.NewTimestamp(p.CreatedAt),
		})
	}

	disciplinaries, err := GetStaffDisciplinaries(ctx, userID, true)

	if err != nil {
		return types.StaffMember{}, err
	}

	resolved := resolveWithDisciplinaries(sp, disciplinaries)

	user, err := GetPlatformUser(ctx, userID)

	if err != nil {
		return types.StaffMember{}, err
	}

	return types.StaffMember{
		UserID:          userID,
		User:            user,
		Positions:       positions,
		Disciplinaries:  disciplinaries,
		PermOverrides:   nonNil(permOverrides),
		ResolvedPerms:   PermStrings(resolved),
		NoAutosync:      noAutosync,
		Unaccounted:     unaccounted,
		MfaVerified:     mfaVerified,
		CreatedAt:       types.NewTimestamp(createdAt),
		StaffPermission: sp,
	}, nil
}

// resolveWithDisciplinaries applies disciplinary permission limits on top of a
// member's positions.
//
// Each disciplinary is pushed as a synthetic position at index 0, which outranks
// every real position. A NON-additory disciplinary additionally drops every
// position that is not itself a synthetic one added so far, i.e. it replaces the
// member's permissions rather than intersecting with them.
func resolveWithDisciplinaries(sp perms.StaffPermissions, disciplinaries []types.StaffDisciplinary) []perms.Permission {
	if len(disciplinaries) == 0 {
		return sp.Resolve()
	}

	virtual := perms.StaffPermissions{
		UserPositions: append([]perms.PartialStaffPosition(nil), sp.UserPositions...),
		PermOverrides: sp.PermOverrides,
	}

	var addedIDs []string

	for _, disc := range disciplinaries {
		virtual.UserPositions = append(virtual.UserPositions, perms.PartialStaffPosition{
			ID:    disc.ID,
			Index: 0,
			Perms: perms.PFSS(disc.Type.PermLimits),
		})

		addedIDs = append(addedIDs, disc.ID)

		if !disc.Type.Additory {
			retained := virtual.UserPositions[:0]

			for _, pos := range virtual.UserPositions {
				if slicesContains(addedIDs, pos.ID) {
					retained = append(retained, pos)
				}
			}

			virtual.UserPositions = retained
		}
	}

	return virtual.Resolve()
}

func slicesContains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}

	return false
}

// nonNil keeps empty arrays encoding as [] rather than null, which is what serde
// does for an empty Vec.
func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}

	return in
}

func nonNilLinks(in []types.Link) []types.Link {
	if in == nil {
		return []types.Link{}
	}

	return in
}
