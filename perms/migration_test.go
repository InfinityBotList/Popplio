package perms

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The migration in exp/rewrite/flatperms.sql and the Legacy lists in the catalogues are
// two statements of the same mapping. These tests keep them from drifting: a
// permission renamed in one place but not the other is caught here rather than
// by a staff member who has quietly lost access.

const migrationPath = "../exp/rewrite/flatperms.sql"

var (
	// ('staff', 'rpc.Claim', 'review_bots')
	tupleRe = regexp.MustCompile(`\('(staff|entity)',\s*'([^']+)',\s*'([^']+)'\)`)
	// SELECT 'staff', 'rpc.*', p FROM unnest(ARRAY[...]) AS p
	wildcardRe = regexp.MustCompile(`(?s)SELECT '(staff|entity)', '([^']+)', p\s*FROM unnest\(ARRAY\[(.*?)\]\)`)
	quotedRe   = regexp.MustCompile(`'([^']+)'`)
)

func readMigration(t *testing.T) string {
	t.Helper()

	b, err := os.ReadFile(migrationPath)

	if err != nil {
		t.Fatalf("reading %s: %v", migrationPath, err)
	}

	return string(b)
}

// retired reports whether the migration drops a permission on purpose, i.e.
// lists it in retired_perm as a ('domain', 'name') pair rather than mapping it.
func retired(sql, domain, perm string) bool {
	body := sql

	if _, after, ok := strings.Cut(sql, "INSERT INTO retired_perm"); ok {
		body, _, _ = strings.Cut(after, ";")
	} else {
		return false
	}

	return strings.Contains(body, "('"+domain+"', '"+perm+"')")
}

func catalogueFor(domain string) *Catalogue {
	if domain == "staff" {
		return Staff
	}

	return Entity
}

// TestMigrationTargetsAreDeclared checks that everything the migration writes
// into the database is a permission the code knows about.
func TestMigrationTargetsAreDeclared(t *testing.T) {
	sql := readMigration(t)

	for _, m := range tupleRe.FindAllStringSubmatch(sql, -1) {
		domain, old, next := m[1], m[2], m[3]

		if _, ok := catalogueFor(domain).Lookup(Perm(next)); !ok {
			t.Errorf("migration maps %s permission %q to %q, which no catalogue declares", domain, old, next)
		}
	}

	for _, m := range wildcardRe.FindAllStringSubmatch(sql, -1) {
		domain, old, body := m[1], m[2], m[3]

		for _, q := range quotedRe.FindAllStringSubmatch(body, -1) {
			if _, ok := catalogueFor(domain).Lookup(Perm(q[1])); !ok {
				t.Errorf("migration expands %s wildcard %q to %q, which no catalogue declares", domain, old, q[1])
			}
		}
	}
}

// TestEveryLegacyPermissionIsMigrated checks the other direction: every old
// permission the catalogues claim to replace is actually handled by the SQL,
// and lands on the permission the catalogue says it should.
func TestEveryLegacyPermissionIsMigrated(t *testing.T) {
	sql := readMigration(t)

	mapped := map[string]map[string]string{
		"staff":  {},
		"entity": {},
	}

	for _, m := range tupleRe.FindAllStringSubmatch(sql, -1) {
		mapped[m[1]][m[2]] = m[3]
	}

	for domain, cat := range map[string]*Catalogue{"staff": Staff, "entity": Entity} {
		for _, d := range cat.Definitions() {
			for _, old := range d.Legacy {
				got, ok := mapped[domain][old]

				if !ok {
					t.Errorf("%s: %q says it replaces %q, but the migration does not map it", domain, d.ID, old)
					continue
				}

				if got != string(d.ID) {
					t.Errorf("%s: %q says it replaces %q, but the migration maps it to %q", domain, d.ID, old, got)
				}
			}
		}
	}
}

// TestMigrationCoversTheOldVocabulary pins the permissions that were actually in
// use, so that dropping one from the migration is a deliberate act.
//
// The list is every distinct value found across staff_positions.perms,
// staff_members.perm_overrides, staff_disciplinary_types.perm_limits,
// team_members.flags and api_sessions.perm_limits before the migration ran.
func TestMigrationCoversTheOldVocabulary(t *testing.T) {
	sql := readMigration(t)

	handled := func(domain, perm string) bool {
		perm = strings.TrimPrefix(perm, "~")

		for _, m := range tupleRe.FindAllStringSubmatch(sql, -1) {
			if m[1] == domain && m[2] == perm {
				return true
			}
		}

		for _, m := range wildcardRe.FindAllStringSubmatch(sql, -1) {
			if m[1] == domain && m[2] == perm {
				return true
			}
		}

		// Retiring a permission is handling it: the migration drops it on
		// purpose rather than leaving it to the unmapped NOTICE.
		if retired(sql, domain, perm) {
			return true
		}

		// Wildcards listed together in one unnest, e.g. shop_items.* and
		// friends, appear as quoted names in the wildcard section.
		return strings.Contains(sql, "'"+perm+"'")
	}

	staffVocab := []string{
		"global.*", "global.view", "global.view_sensitive", "arcadia.*",
		"rpc.*", "rpc.Claim", "rpc.Unclaim", "rpc.Approve", "rpc.Deny", "rpc.Unverify",
		"~rpc.PremiumAdd", "~rpc.PremiumRemove", "~rpc.VoteBanAdd", "~rpc.VoteBanRemove",
		"rpc_logs.*", "rpc_logs.view", "apps.*", "apps.view",
		"staff_members.*", "staff_members.view", "staff_positions.*", "staff_positions.view",
		"staff_disciplinary_types.*", "persepolis.*", "persepolis.view_onboarding_responses",
		"shop.view", "shop_items.*", "shop_items.view", "shop_item_benefits.*",
		"shop_item_benefits.update", "shop_item_benefits.view", "vote_credit_tiers.*",
		"bot_whitelist.*", "partners.*", "popplio.*", "popplio_staging.*", "borealis.*",
		"cdn.list_scopes", "cdn.list", "cdn.add_file", "cdn.upload_chunk",
		"cdn#ibl@main.*", "cdn#ibl@main.list", "cdn#ibl@main.read_file",
		"developer.marker", "lead_developer.marker", "human_resources.marker",
		"bot_reviewer.marker", "service_account.marker", "dt.marker",
	}

	for _, perm := range staffVocab {
		if !handled("staff", perm) {
			t.Errorf("staff permission %q was in use but the migration does not handle it", perm)
		}
	}

	entityVocab := []string{
		"global.*", "global.add", "global.edit", "global.delete", "global.resubmit",
		"global.set_vanity", "global.request_cert", "global.get_webhooks", "global.edit_webhooks",
		"global.test_webhooks", "global.get_webhook_logs", "global.delete_webhook_logs",
		"global.upload_assets", "global.delete_assets", "global.create_owner_review",
		"global.edit_owner_review", "global.delete_owner_review", "global.redeem_vote_credits",
		"bot.*", "bot.add", "bot.edit", "bot.delete", "bot.resubmit", "bot.request_cert",
		"bot.set_vanity", "bot.get_webhooks", "bot.edit_webhooks", "bot.test_webhooks",
		"bot.get_webhook_logs", "bot.delete_webhook_logs", "bot.upload_assets", "bot.delete_assets",
		"bot.create_owner_review", "bot.edit_owner_review", "bot.delete_owner_review",
		"bot.redeem_vote_credits", "bot.view_api_tokens", "bot.reset_api_tokens",
		"server.*", "team.*", "team.edit", "team.edit_webhooks",
		"team_member.*", "team_member.add", "team_member.edit", "team_member.delete", "team_member.remove",
	}

	for _, perm := range entityVocab {
		if !handled("entity", perm) {
			t.Errorf("entity permission %q was in use but the migration does not handle it", perm)
		}
	}
}

// Borealis was removed from the platform in the port, so its permission gated
// nothing. It must be gone from the catalogue and dropped by the migration
// rather than mapped onto something else.
func TestBorealisIsRetired(t *testing.T) {
	for _, d := range Staff.Definitions() {
		if strings.Contains(strings.ToLower(string(d.ID)), "boreal") {
			t.Errorf("the staff catalogue still declares %q", d.ID)
		}

		for _, old := range d.Legacy {
			if strings.Contains(strings.ToLower(old), "boreal") {
				t.Errorf("%q still claims to replace %q", d.ID, old)
			}
		}
	}

	sql := readMigration(t)

	if !retired(sql, "staff", "borealis.*") {
		t.Error("the migration should list borealis.* as retired")
	}

	if strings.Contains(sql, "use_borealis") {
		t.Error("the migration still writes use_borealis into the database")
	}
}
