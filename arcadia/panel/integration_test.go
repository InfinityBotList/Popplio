package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests drive the real POST / handler against a real Postgres, which is
// the only way to catch a mis-typed column now that sqlx's compile-time query
// verification is gone (§13). They skip unless a database is configured:
//
//	ARCADIA_TEST_DATABASE_URL=postgres://user:pass@127.0.0.1/arcadia_test \
//	  go test -ldflags=-checklinkname=0 ./arcadia/panel/
//
// Point this at a SCRATCH database: the fixtures insert and delete rows.
//
// Operations that resolve a dovewing user (Hello, GetUser, BotQueue,
// SearchEntitys, UpdateStaffMembers) are not covered here because dovewing needs
// Redis and a live Discord session. Everything reachable with just Postgres is.

// dbOrSkip connects the shared pool, or skips.
func dbOrSkip(t *testing.T) {
	t.Helper()

	if state.Pool != nil {
		return
	}

	url := os.Getenv("ARCADIA_TEST_DATABASE_URL")

	if url == "" {
		t.Skip("ARCADIA_TEST_DATABASE_URL is not set; skipping integration tests")
	}

	pool, err := pgxpool.New(context.Background(), url)

	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	state.Pool = pool
}

// fixture is a seeded staff member with a session.
type fixture struct {
	UserID     string
	Token      string
	PositionID string
}

// seedStaff creates a user, a staff position carrying perms, a staff member and
// an authchain session in the given state. Everything is removed on cleanup.
func seedStaff(t *testing.T, perms []string, sessionState string) fixture {
	t.Helper()
	dbOrSkip(t)

	ctx := context.Background()

	f := fixture{
		UserID: fmt.Sprintf("9%014d", len(t.Name())*7919+len(perms)),
		Token:  impls.GenRandom(64),
	}

	// A unique-per-test user id keeps parallel-ish runs from colliding.
	f.UserID = fmt.Sprintf("test_%s", impls.GenRandom(12))

	cleanup := func() {
		state.Pool.Exec(ctx, "DELETE FROM staffpanel__authchain WHERE user_id = $1", f.UserID)
		state.Pool.Exec(ctx, "DELETE FROM staff_members WHERE user_id = $1", f.UserID)
		state.Pool.Exec(ctx, "DELETE FROM rpc_logs WHERE user_id = $1", f.UserID)
		state.Pool.Exec(ctx, "DELETE FROM users WHERE user_id = $1", f.UserID)

		if f.PositionID != "" {
			state.Pool.Exec(ctx, "DELETE FROM staff_positions WHERE id = $1", f.PositionID)
		}
	}

	cleanup()
	t.Cleanup(cleanup)

	_, err := state.Pool.Exec(ctx, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", f.UserID, impls.GenRandom(64))

	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	err = state.Pool.QueryRow(ctx,
		"INSERT INTO staff_positions (name, role_id, perms, index) VALUES ($1, $2, $3, $4) RETURNING id::text",
		"test_"+impls.GenRandom(8), "0", perms, 50,
	).Scan(&f.PositionID)

	if err != nil {
		t.Fatalf("seed position: %v", err)
	}

	_, err = state.Pool.Exec(ctx,
		"INSERT INTO staff_members (user_id, positions) VALUES ($1, ARRAY[$2::uuid])",
		f.UserID, f.PositionID)

	if err != nil {
		t.Fatalf("seed staff member: %v", err)
	}

	_, err = state.Pool.Exec(ctx,
		"INSERT INTO staffpanel__authchain (user_id, token, popplio_token, state) VALUES ($1, $2, $3, $4)",
		f.UserID, f.Token, impls.GenRandom(64), sessionState)

	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	return f
}

// post drives a PanelQuery through the real HTTP handler.
func post(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	handler(t).ServeHTTP(rec, req)

	return rec
}

// The session GC and the two auth validators are the gate in front of almost
// every operation, and the panel frontend matches on their exact messages.
func TestAuthValidators(t *testing.T) {
	dbOrSkip(t)

	ctx := context.Background()

	t.Run("unknown token is identityExpired", func(t *testing.T) {
		_, err := impls.CheckAuthInsecure(ctx, "nope-"+impls.GenRandom(20))

		if err == nil || err.Error() != "identityExpired" {
			t.Errorf("err = %v, want identityExpired", err)
		}
	})

	t.Run("pending session fails CheckAuth but passes insecure", func(t *testing.T) {
		f := seedStaff(t, []string{"rpc.Claim"}, "pending")

		data, err := impls.CheckAuthInsecure(ctx, f.Token)

		if err != nil {
			t.Fatalf("CheckAuthInsecure: %v", err)
		}

		if data.UserID != f.UserID || data.State != "pending" {
			t.Errorf("auth data = %+v", data)
		}

		if _, err := impls.CheckAuth(ctx, f.Token); err == nil || err.Error() != "sessionNotActive" {
			t.Errorf("CheckAuth err = %v, want sessionNotActive", err)
		}
	})

	t.Run("active session passes both", func(t *testing.T) {
		f := seedStaff(t, []string{"rpc.Claim"}, "active")

		if _, err := impls.CheckAuth(ctx, f.Token); err != nil {
			t.Errorf("CheckAuth: %v", err)
		}
	})

	t.Run("staff member with no positions is identityExpired", func(t *testing.T) {
		f := seedStaff(t, []string{}, "active")

		if _, err := state.Pool.Exec(ctx, "UPDATE staff_members SET positions = '{}' WHERE user_id = $1", f.UserID); err != nil {
			t.Fatalf("blank positions: %v", err)
		}

		if _, err := impls.CheckAuth(ctx, f.Token); err == nil || err.Error() != "identityExpired" {
			t.Errorf("err = %v, want identityExpired", err)
		}
	})

	t.Run("session GC removes stale pending sessions", func(t *testing.T) {
		f := seedStaff(t, []string{}, "pending")

		// Age the session past the 5 minute pending window.
		_, err := state.Pool.Exec(ctx,
			"UPDATE staffpanel__authchain SET created_at = NOW() - INTERVAL '10 minutes' WHERE token = $1", f.Token)

		if err != nil {
			t.Fatalf("age session: %v", err)
		}

		// Any auth call runs the GC first.
		impls.CheckAuthInsecure(ctx, "trigger-gc")

		var count int64

		err = state.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM staffpanel__authchain WHERE token = $1", f.Token).Scan(&count)

		if err != nil {
			t.Fatalf("count: %v", err)
		}

		if count != 0 {
			t.Error("stale pending session survived the GC")
		}
	})
}

// Auth failures surface as 500 with a bare string, because Error::new defaults to
// 500. The frontend matches on the body text, so both parts matter.
func TestAuthFailureShape(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"BaseAnalytics":{"login_token":"definitely-not-a-token"}}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}

	if got := rec.Body.String(); got != "identityExpired" {
		t.Errorf("body = %q, want identityExpired", got)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
}

// Logout takes no auth at all and returns the affected row count as bare text.
func TestLogoutReturnsRowCountAsText(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"Authorize":{"version":5,"action":{"Logout":{"login_token":%q}}}}`, f.Token))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	if got := rec.Body.String(); got != "1" {
		t.Errorf("body = %q, want \"1\"", got)
	}

	// A second logout affects no rows.
	rec = post(t, fmt.Sprintf(`{"Authorize":{"version":5,"action":{"Logout":{"login_token":%q}}}}`, f.Token))

	if got := rec.Body.String(); got != "0" {
		t.Errorf("second logout body = %q, want \"0\"", got)
	}
}

// Hello validates auth BEFORE the version, so a bad version on a bad token still
// reports the auth failure.
func TestHelloChecksAuthBeforeVersion(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"Hello":{"login_token":"bad","version":999}}`)

	if got := rec.Body.String(); got != "identityExpired" {
		t.Errorf("body = %q, want identityExpired (auth is checked first)", got)
	}
}

// Authorize checks the version before anything else and needs no session.
func TestAuthorizeVersionGate(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"Authorize":{"version":4,"action":{"Begin":{"scope":"s","redirect_url":"r"}}}}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}

	if got := rec.Body.String(); got != "Invalid version" {
		t.Errorf("body = %q, want Invalid version", got)
	}
}

// §14.3: every guarded operation must be tested allowed and denied. These all
// resolve permissions through the light path, so they need no Discord.
func TestPermissionGates(t *testing.T) {
	dbOrSkip(t)

	cases := []struct {
		name       string
		perm       string
		body       func(token string) string
		wantDenied string
	}{
		{
			name:       "GetRpcLogEntries",
			perm:       "rpc_logs.view",
			body:       func(tok string) string { return fmt.Sprintf(`{"GetRpcLogEntries":{"login_token":%q}}`, tok) },
			wantDenied: "You do not have permission to view rpc logs [rpc_logs.view]",
		},
		{
			name: "UpdateShopCoupons/List",
			perm: "shop_coupons.list",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateShopCoupons":{"login_token":%q,"action":"List"}}`, tok)
			},
			wantDenied: "You do not have permission to list shop coupons [shop_coupons.list]",
		},
		{
			name: "UpdateVoteCreditTiers/DeleteTier",
			perm: "vote_credit_tiers.delete",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateVoteCreditTiers":{"login_token":%q,"action":{"DeleteTier":{"id":"nope"}}}}`, tok)
			},
			wantDenied: "You do not have permission to delete vote credit tiers [vote_credit_tiers.delete]",
		},
		{
			name: "UpdateBotWhitelist/Delete",
			perm: "bot_whitelist.delete",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateBotWhitelist":{"login_token":%q,"action":{"Delete":{"bot_id":"nope"}}}}`, tok)
			},
			wantDenied: "You do not have permission to delete bot whitelist entries (bot_whitelist.delete)",
		},
		{
			name: "UpdateBlog/DeleteEntry",
			perm: "blog.delete_entry",
			body: func(tok string) string {
				return fmt.Sprintf(`{"UpdateBlog":{"login_token":%q,"action":{"DeleteEntry":{"itag":"00000000-0000-0000-0000-000000000000"}}}}`, tok)
			},
			wantDenied: "You do not have permission to delete blog entries [blog.delete_entry]",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name+"/denied", func(t *testing.T) {
			f := seedStaff(t, []string{"some.unrelated"}, "active")

			rec := post(t, tt.body(f.Token))

			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
			}

			if got := rec.Body.String(); got != tt.wantDenied {
				t.Errorf("body = %q\nwant %q", got, tt.wantDenied)
			}
		})

		t.Run(tt.name+"/allowed", func(t *testing.T) {
			f := seedStaff(t, []string{tt.perm}, "active")

			rec := post(t, tt.body(f.Token))

			// Allowed means "got past the gate": anything but the 403 denial.
			if rec.Code == http.StatusForbidden && rec.Body.String() == tt.wantDenied {
				t.Errorf("permission %q was granted but the request was still denied", tt.perm)
			}
		})
	}
}

// The vote credit tier position dedup loop is load-bearing for ordering.
//
// vote_credit_tiers.position carries a UNIQUE constraint that is DEFERRABLE
// INITIALLY DEFERRED, which is what lets the loop insert onto an occupied
// position and tidy up before COMMIT.
//
// The loop handles ONE existing occupant correctly and is broken for two or
// more. See CONFORMANCE.md a/16 - this test pins both halves so the fix is a
// deliberate, visible change.
func TestVoteCreditTierDedupLoop(t *testing.T) {
	f := seedStaff(t, []string{"vote_credit_tiers.create"}, "active")

	ctx := context.Background()

	create := func(id string) *httptest.ResponseRecorder {
		return post(t, fmt.Sprintf(
			`{"UpdateVoteCreditTiers":{"login_token":%q,"action":{"CreateTier":{"id":%q,"target_type":"bot","position":1,"cents":0.5,"votes":100}}}}`,
			f.Token, id))
	}

	ids := []string{"tier_a_" + impls.GenRandom(6), "tier_b_" + impls.GenRandom(6), "tier_c_" + impls.GenRandom(6)}

	t.Cleanup(func() {
		for _, id := range ids {
			state.Pool.Exec(ctx, "DELETE FROM vote_credit_tiers WHERE id = $1", id)
		}
	})

	// One existing occupant: the loop shifts it down and both end up unique.
	for _, id := range ids[:2] {
		if rec := create(id); rec.Code != http.StatusNoContent {
			t.Fatalf("create %s: status = %d, body = %s", id, rec.Code, rec.Body.String())
		}
	}

	positions := tierPositions(t, ids[:2])

	if positions[ids[0]] != 2 || positions[ids[1]] != 1 {
		t.Errorf("after two creates positions = %v, want the first tier pushed to 2 and the second at 1", positions)
	}

	// Two existing occupants: the loop moves the row at position 1 to position 2,
	// which is already taken, and then resumes checking at position 3 - never
	// re-examining the collision it just created. The deferred constraint fires
	// at COMMIT and the panel gets a raw Postgres error.
	rec := create(ids[2])

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("third create: status = %d, want 500 (reproduced upstream bug); body = %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), "duplicate key value violates unique constraint") {
		t.Errorf("third create body = %q, want the deferred unique violation", rec.Body.String())
	}
}

// tierPositions reads back the positions of the given tiers.
func tierPositions(t *testing.T, ids []string) map[string]int32 {
	t.Helper()

	rows, err := state.Pool.Query(context.Background(),
		"SELECT id, position FROM vote_credit_tiers WHERE id = ANY($1)", ids)

	if err != nil {
		t.Fatalf("query positions: %v", err)
	}

	defer rows.Close()

	out := map[string]int32{}

	for rows.Next() {
		var (
			id  string
			pos int32
		)

		if err := rows.Scan(&id, &pos); err != nil {
			t.Fatalf("scan: %v", err)
		}

		out[id] = pos
	}

	return out
}

// The shop coupon validations reject a null max_uses, which is the reproduced
// upstream bug that makes "unlimited uses" unreachable (CONFORMANCE.md a/2).
// This test pins the buggy behaviour so the fix is a deliberate, visible change.
func TestShopCouponNullValidationIsReproduced(t *testing.T) {
	f := seedStaff(t, []string{"shop_coupons.create"}, "active")

	body := fmt.Sprintf(`{"UpdateShopCoupons":{"login_token":%q,"action":{"Create":{"id":"x","code":"c","public":true,"max_uses":null,"reuse_wait_duration":1,"expiry":1,"applicable_items":[],"cents":null,"requirements":[],"allowed_users":[],"usable":true,"target_types":[]}}}}`, f.Token)

	rec := post(t, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}

	if got := rec.Body.String(); got != "Max uses must be greater than 0" {
		t.Errorf("body = %q, want %q", got, "Max uses must be greater than 0")
	}
}

// ListPositions needs no permission and returns a JSON array, exercising the
// staff_positions read path and the Link JSONB decode.
func TestListPositionsIsOpenToAllStaff(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"UpdateStaffPositions":{"login_token":%q,"action":"ListPositions"}}`, f.Token))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var positions []types.StaffPosition

	if err := json.Unmarshal(rec.Body.Bytes(), &positions); err != nil {
		t.Fatalf("response is not a StaffPosition array: %v\nbody: %s", err, rec.Body.String())
	}

	var found bool

	for _, p := range positions {
		if p.ID == f.PositionID {
			found = true

			if p.CorrespondingRoles == nil {
				t.Error("corresponding_roles decoded as null, want []")
			}
		}
	}

	if !found {
		t.Error("the seeded position is missing from ListPositions")
	}
}

// BaseAnalytics exercises four grouped-count queries and the hardcoded zero.
func TestBaseAnalytics(t *testing.T) {
	f := seedStaff(t, []string{}, "active")

	rec := post(t, fmt.Sprintf(`{"BaseAnalytics":{"login_token":%q}}`, f.Token))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var analytics types.BaseAnalytics

	if err := json.Unmarshal(rec.Body.Bytes(), &analytics); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, rec.Body.String())
	}

	if analytics.ChangelogsCount != 0 {
		t.Errorf("changelogs_count = %d, want 0 (hardcoded)", analytics.ChangelogsCount)
	}

	if analytics.TotalUsers < 1 {
		t.Errorf("total_users = %d, want at least the seeded user", analytics.TotalUsers)
	}

	// Every count map must encode as an object, never null.
	if analytics.BotCounts == nil || analytics.ServerCounts == nil || analytics.TicketCounts == nil {
		t.Error("a count map decoded as null")
	}
}

// UpdateChangelog is a hard stub: 403 regardless of input or authentication.
func TestChangelogStubIgnoresAuth(t *testing.T) {
	dbOrSkip(t)

	rec := post(t, `{"UpdateChangelog":{"login_token":"not-a-real-token","action":"ListEntries"}}`)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}

	if got := rec.Body.String(); got != "You do not have permission to create changelog entries [not implemented]" {
		t.Errorf("body = %q", got)
	}
}
