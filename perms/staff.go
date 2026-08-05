package perms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"popplio/state"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
)

// staffQuery loads a staff member's extra permissions and every role they hold
// in one round trip, ordered by rank so the most senior role comes first.
//
// The join onto dovewing's user cache is the read side of [ErrBotAccount]: it
// costs nothing here and does not need Discord to be reachable, so a bot account
// resolves to no permissions even while the write paths that should have stopped
// it are unavailable.
const staffQuery = `SELECT
	sm.perm_overrides,
	COALESCE((
		SELECT json_agg(json_build_object('id', sp.id::text, 'name', sp.name, 'index', sp.index, 'perms', sp.perms) ORDER BY sp.index)
		FROM staff_positions sp
		WHERE sp.id = ANY(sm.positions)
	), '[]'::json),
	COALESCE(iuc.bot, false)
FROM staff_members sm
LEFT JOIN internal_user_cache__discord iuc ON iuc.id = sm.user_id
WHERE sm.user_id = $1`

// StaffGrants is where a staff member's permissions come from: the roles they
// hold, which the Discord resync keeps in step with the staff server, plus the
// extras granted to them individually.
type StaffGrants struct {
	Roles  []Role `json:"roles"`
	Extras []Perm `json:"extras"`

	// ConfigOwner marks an instance owner from the config. See [IsConfigOwner]:
	// they hold everything and outrank everyone, whatever the database says.
	ConfigOwner bool `json:"config_owner"`

	// BotAccount marks a Discord bot account. A bot holds no staff permissions
	// whatever the database says — see [ErrBotAccount].
	BotAccount bool `json:"bot_account"`
}

// Resolve is every permission the member has.
func (g StaffGrants) Resolve() Set {
	// A bot resolves to nothing before anything else is considered, including
	// the owners list: a bot named there is a misconfiguration, not a grant.
	if g.BotAccount {
		return Staff.NewSet()
	}

	set := Staff.Resolve(g.Roles, g.Extras...)

	if g.ConfigOwner {
		return set.With(StaffAdministrator)
	}

	return set
}

// TopRole is the member's most senior role, which is what rank checks compare
// against. The second return is false for a member with no roles at all.
func (g StaffGrants) TopRole() (Role, bool) {
	if len(g.Roles) == 0 {
		return Role{}, false
	}

	top := g.Roles[0]

	for _, r := range g.Roles[1:] {
		if r.Index < top.Index {
			top = r
		}
	}

	return top, true
}

// Rank is the member's seniority: the index of their most senior role, lower
// being more senior. A member with no roles ranks below everyone; an instance
// owner outranks everyone.
func (g StaffGrants) Rank() int32 {
	// A bot holds no permissions, so it has no standing to outrank anyone.
	if g.BotAccount {
		return NoRank
	}

	if g.ConfigOwner {
		return OwnerRank
	}

	top, ok := g.TopRole()

	if !ok {
		return NoRank
	}

	return top.Index
}

// NoRank is the rank of someone who holds no roles. Every real role outranks
// it, so a rank check against it always fails closed.
const NoRank int32 = 1<<31 - 1

type staffRoleJSON struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Index int32    `json:"index"`
	Perms []string `json:"perms"`
}

// LoadStaff returns where a staff member's permissions come from.
//
// Use it when the roles themselves matter — the panel and the staff bot show
// them, and disciplinaries are applied on top. For a plain access check use
// [StaffPerms].
//
// A user with no staff_members row is not staff and the error wraps
// pgx.ErrNoRows.
func LoadStaff(ctx context.Context, userID string) (StaffGrants, error) {
	var (
		extras []string
		roles  []byte
		bot    bool
	)

	owner := IsConfigOwner(userID)

	err := state.Pool.QueryRow(ctx, staffQuery, userID).Scan(&extras, &roles, &bot)

	if err != nil {
		// An owner is an owner before the resync has ever run, so a missing row
		// is not an obstacle for them. Being a bot still is, so that branch pays
		// for the one lookup the main query would have done for it.
		if owner && errors.Is(err, pgx.ErrNoRows) {
			return StaffGrants{ConfigOwner: true, BotAccount: cachedBotFlag(ctx, userID)}, nil
		}

		return StaffGrants{}, fmt.Errorf("error while getting staff perms of user %s: %w", userID, err)
	}

	var rows []staffRoleJSON

	if err := json.Unmarshal(roles, &rows); err != nil {
		return StaffGrants{}, fmt.Errorf("error while getting staff roles of user %s: %w", userID, err)
	}

	g := StaffGrants{
		Roles:       make([]Role, 0, len(rows)),
		Extras:      ParseStrings(extras),
		ConfigOwner: owner,
		BotAccount:  bot,
	}

	for _, row := range rows {
		g.Roles = append(g.Roles, Role{
			ID:    row.ID,
			Name:  row.Name,
			Index: row.Index,
			Perms: ParseStrings(row.Perms),
		})
	}

	return g, nil
}

// StaffPerms resolves a staff member's permissions.
//
// Disciplinaries are not applied here; this is the light path used by RPC calls
// and most panel operations. The panel's own member view applies them on top.
func StaffPerms(ctx context.Context, userID string) (Set, error) {
	g, err := LoadStaff(ctx, userID)

	if err != nil {
		return Set{}, err
	}

	return g.Resolve(), nil
}

// cachedBotFlag reads what dovewing's user cache already knows about an account,
// without going to Discord for it.
//
// An account nobody has cached yet reads as "not a bot", which is the same
// answer the join in [staffQuery] gives. That is not the guarantee — the write
// paths and [RejectBotAccount] are — it is the cheap check that holds the line
// on every permission read.
func cachedBotFlag(ctx context.Context, userID string) bool {
	var bot bool

	err := state.Pool.QueryRow(ctx, "SELECT bot FROM internal_user_cache__discord WHERE id = $1", userID).Scan(&bot)

	if err != nil {
		return false
	}

	return bot
}

// ErrBotAccount is a bot account being offered staff permissions. Bots hold no
// staff permissions at all: not through a role, not through a direct grant, not
// through the owners list.
//
// A bot is a program with a token that other programs can be given, and the
// staff model is built on a person being accountable for what their permissions
// did. Nothing in the platform needs a bot to be staff either — the staff bot
// and the panel act under a staff member's identity, never their own.
var ErrBotAccount = errors.New("bot accounts cannot hold staff permissions")

// RejectBotAccount fails with [ErrBotAccount] if a user is a Discord bot.
//
// This is the write side of the rule, for the paths that hand out permissions,
// and it is deliberately the expensive check: it resolves through dovewing
// (Redis, then the user cache table, then Discord itself), so a bot that has
// never been seen before is still recognised. A lookup that cannot be resolved
// fails closed, since being unable to tell what an account is is not a reason to
// give it staff permissions.
//
// The read paths do not call this. They rely on the cached flag [LoadStaff]
// already carries, so permission checks stay one query and keep working while
// Discord does not.
func RejectBotAccount(ctx context.Context, userID string) error {
	user, err := dovewing.GetUser(ctx, userID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("could not check whether %s is a bot: %w", userID, err)
	}

	if user.Bot {
		return ErrBotAccount
	}

	return nil
}
