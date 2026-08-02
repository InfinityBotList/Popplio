# Arcadia → Go port: conformance notes

Arcadia (Rust: axum + sqlx + serenity/poise, ~12,800 LOC) ported into Popplio.
The wire format is frozen — a live SvelteKit staff panel and other Infinity List
services already speak it — so the default everywhere was to reproduce upstream
behaviour byte-for-byte, including its bugs.

Layout:

| Package | Contents |
|---|---|
| `arcadia/types` | Wire DTOs and the tagged-union codec (§3) |
| `arcadia/impls` | Auth/session, permissions, entity managers, dovewing adapter, Discord helpers |
| `arcadia/rpc` | The shared action layer: pipeline + all 18 methods (§7) |
| `arcadia/panel` | Custom net/http server, middleware, dispatcher, all 25 operations (§4, §5) |
| `arcadia/cdnpath` | CDN name/path validators and the granular CDN permission check |
| `arcadia/tasks` | The 12 background tasks and their runner (§12) |
| `arcadia/bot` | Discord command framework, commands, events, guards (§11) |
| `arcadia/conformance` | Frozen-string tests |

---

## (a) Upstream bugs and quirks

Policy: **reproduce all**, document each, and offer the fix as a separate opt-in
patch. Four internal-only defects were fixed because they change nothing a panel
user or staff member can observe; each is called out as FIXED below.

### Reproduced

**1. Partner avatar path mismatch (§5.14)** — `arcadia/panel/ops_content.go`
Create and Update validate the avatar at `<main_scope>/avatars/partners/<id>.webp`,
but Delete removes `<main_scope>/partners/<id>.webp` — a different directory.
Deleting a partner therefore never removes its avatar, and the asset_cleaner task
later reaps it. Reproduced exactly.
*Patch:* change the Delete path to `avatars/partners/`. Deferred because the
already-orphaned files under the old path would then be missed.

**2. Shop coupon Option validation makes "unlimited" unreachable (§5.23)** — `arcadia/panel/ops_shop.go`
`max_uses`, `reuse_wait_duration` and `expiry` are validated as
`value.unwrap_or(0) <= 0 → reject`. A `null` therefore becomes `0` and **fails**,
so the "unlimited uses" and "never expires" cases the DTO comments describe
cannot be created through this endpoint at all. This is a real bug that silently
narrows the product.
*Patch (one line each, in `validateCoupon`):*
```go
if action.MaxUses != nil && *action.MaxUses <= 0 { … }
```
i.e. treat `None` as "no constraint". **Recommended**, but it changes what the
panel accepts, so it needs a product decision.

**3. `topreviewer_sync` runs with `LIMIT 0` (§12)** — `arcadia/tasks/discord.go`
The weekly job strips the top-reviewer role from every main-guild member and then
re-grants it to the top `0` reviewers — i.e. to nobody. The Discord `refresh`
command runs the same query with `LIMIT 3`. **Explicitly confirmed to keep as-is.**
*Patch:* change `const limit = 0` to `3`. Visible effect: staff would keep a role
they currently lose every 7 days.

**4. Chunk-id retry loop tests the same id ten times (§5.10)** — `arcadia/panel/ops_cdn.go`
The id is generated once, outside the loop, so the ten attempts all check the same
key. With a 32-character alphanumeric id a collision is not a practical concern;
the loop is simply dead code.
*Patch:* move `impls.GenRandom(32)` inside the loop.

**5. `PopplioStaff` sends the request PATH as `X-Forwarded-For` (§5.25)** — `arcadia/panel/ops_core.go`
Not a typo we can safely "fix": Popplio may key on it. Reproduced, and flagged
loudly here. Worth confirming with whoever owns the Popplio staff endpoints
whether anything reads it; if not, it should be dropped or set to the caller's IP.

**6. `Claim`'s `testbot` branch is dead (§7.2)** — `arcadia/rpc/methods.go`
`type != "pending"` is rejected immediately above, so `type == "testbot"` can
never be reached. Kept so the code still reads like the source. (`Unclaim` checks
`testbot` *first*, so there the branch is live.)

**7. Disciplinary type `created_at` is the disciplinary's, not the type's (§8.2)** — `arcadia/impls/auth.go`
`StaffDisciplinaryType.created_at` is populated from the *disciplinary* row. The
panel therefore shows a type as having been created when the punishment was
issued. Wire-visible, so reproduced.

**8. "testing" corresponding-server is accepted but ignored (§12.1)** — `arcadia/tasks/staffresync.go`
The panel's position validator (§5.17) accepts link names `main`, `testing` and
`staff`, but `modify_corresponding_roles` only handles `main` and `staff` and
warns-and-skips anything else. A position configured with a `testing`
corresponding role validates cleanly and then silently never syncs. Reproduced;
the fix is a one-line `case "testing":` in `collectCorrespondingRoles`.

**9. `Unverify`'s mod-log embed has an empty field name** — `arcadia/rpc/methods.go`
**Newly found during the port.** The third embed field is built with an empty
`name`, which the Discord API rejects (field names must be 1–256 characters). The
embed post therefore fails, the error propagates, and `Unverify` reports failure
*after* having already flipped the bot to `pending`. Reproduced faithfully; this
one is worth fixing soon — the DB write is not rolled back.

**10. `Approve` calls Borealis and posts to Discord inside the transaction (§7.2)** — `arcadia/rpc/methods.go`
An HTTP round trip and a Discord message both happen before `COMMIT`, holding the
transaction open across two network calls. Preserved, since restructuring changes
failure semantics (today a failed Borealis call rolls back the approval).

**11. Several error strings begin with a bare space** — `arcadia/rpc/methods.go`
`" does not exist"`, `" is not pending review?"`, `" is in a team. …"` — the
entity name was dropped in an earlier refactor. The panel shows them raw. Frozen
and asserted by `arcadia/conformance`.

**12. Misspelling and inconsistent casing, frozen** —
`"[neeed to delete position]"` (§5.17); `"Invalid OTP Entered"` in `ResetMfaTotp`
vs `"Invalid OTP entered"` in `ActivateSession` (§5.1); `bot_whitelist` permission
messages use `(parentheses)` where every other message uses `[brackets]` (§5.24);
embed titles with intentional-by-accident leading spaces (`" Claimed!"`,
`" Approved!"`, `" Force Deleted!"`).

**13. `Authorize/Begin` does not validate or URL-encode `redirect_url` (§5.1)** — `arcadia/panel/ops_authorize.go`
It is interpolated raw into the Discord OAuth2 URL. `CreateSession` *does* check
the redirect against the allow-list, so the exposure is limited to handing back a
malformed or attacker-chosen login URL to whoever asked for it. Preserved
byte-for-byte; validating it here would be a cheap defence-in-depth improvement.

**14. `UpdateChangelog` is a hard stub (§5.15)** — `arcadia/panel/ops_content.go`
Always 403, regardless of input or authentication. The DTOs still parse.

**15. `UpdateBlog` authenticated twice (§5.16)**
Upstream calls `check_auth` twice with a TODO admitting it is wasteful. Done once
here. No wire difference; the only effect is one fewer round trip and one fewer
run of the session GC.

### Fixed (internal only, no observable change)

**F1. `AddFile` leaked its temp file on hash mismatch (§5.13)** — `arcadia/panel/ops_cdn.go`
Upstream returns early on a SHA-512 mismatch without removing
`/tmp/arcadia-cdn-file…`, leaking a file up to the size of the upload while the
chunks that produced it have already been consumed. Now removed by `defer` on
every exit path. Same status codes, same bodies.

**F2. `japi_updater`'s hour-reset check underflowed (§12)** — `arcadia/tasks/bots.go`
`LAST_REFRESH - now >= 3600` on unsigned integers underflows whenever
`now > LAST_REFRESH`, which is always after the first run, so the 1800-requests
budget reset erratically. Implemented as `now - last >= 3600`.

**F3. `queue`'s previous-page handler underflowed (§11.3)** — `arcadia/bot/interactions.go`
`if current == 0 { current = 0 } current -= 1` underflows on the first page.
Both bounds are clamped.

**F4. `staff_resync` panicked on a cache miss (§12.1)** — `arcadia/tasks/staffresync.go`
The unaccounted-user branch `.unwrap()`s a `member_pos_cache` lookup that can miss
for a user present in the DB but filtered out of the cache. Treated as "no
positions" and logged.

### Not ported

`src/tasks/__toberewritten/uptime.rs` — dead code that does not compile against
the current serenity API. Note that `bots.uptime`, `bots.total_uptime` and
`bots.uptime_last_checked` exist in the schema and are currently unmaintained.

`src/test.rs` (`modaltest`) — a dev-only command never registered in `main.rs`.

---

## (b) Security hardening

Each item below is additive: it rejects input that upstream should never have
accepted, and provably does not change behaviour for legitimate input.

**H1. CDN path containment** — `arcadia/cdnpath/cdnpath.go`, `arcadia/panel/ops_cdn.go`
`validateName`/`validatePath` are pure string checks and are the *only* thing
preventing directory escape from the CDN root. The resolved `asset_path`,
`asset_final_path` and `copy_to` are now additionally checked with
`filepath.Clean` + prefix containment against the scope root.
*Proof of no behaviour change:* any path that stays under the scope root passes
`ContainedInScope` by construction — it can only reject a path that resolves
outside the root, which the validators were already meant to forbid.
Covered by `TestContainedInScope` and `TestValidatePath`.

**H2. `PopplioStaff` proxy path validation** — `arcadia/panel/paths.go`
Upstream checks only that the path starts with `/`, so `//evil.example/x` (a
protocol-relative URL) retargets the request at another host and `..` segments
escape the API base. `safeJoinPopplio` resolves the reference against the base URL
and rejects anything carrying a scheme or host, or resolving outside the base.
*Proof of no behaviour change:* an ordinary absolute path with an optional query
string resolves to exactly the same URL string concatenation produced.

**H3. `generic_cleaner` dynamic SQL** — `arcadia/tasks/cleaners.go`
Upstream interpolates `information_schema` output straight into SQL. Table names
are now filtered to `public` schema, checked against a strict identifier pattern,
and quoted with `pgx.Identifier.Sanitize`. Entity tables and id columns come from
a fixed allow-list. The set of tables acted on is unchanged.

**H4. `asset_cleaner` dynamic SQL** — `arcadia/tasks/cleaners.go`
The table/column pairs come from a hard-coded list (never user input) and only the
id value is parameterised.

**H5. Session tokens use crypto/rand** — `arcadia/impls/crypto.go`
`impls.GenRandom` reads from `crypto/rand` over a full alphanumeric alphabet.
Popplio's existing `eureka/crypto.RandString` was deliberately *not* reused for
this: it draws from `math/rand` seeded with the wall clock and its alphabet omits
digits, which would make panel session tokens and Popplio staff tokens predictable.

**H6. `git commit -m <message>` is never shelled** — `arcadia/panel/ops_cdn.go`
`exec.Command` with separate argv elements, as upstream does. Never `sh -c`.

**H7. `asset_cleaner` dry-run** — `arcadia/tasks/cleaners.go`
The task deletes files off the CDN. `arcadia.asset_cleaner_dry_run` makes it log
what it *would* remove; every deletion is logged either way.

---

## (c) Intentional deviations

**D1. Integration into Popplio.** Per the requester's decision, the port lives
inside Popplio rather than as a standalone service. Consequences:

- **Config.** The Arcadia keys Popplio already had are read from Popplio's config
  rather than duplicated (`database_url` → `meta.postgres_url`, `token` →
  `discord_auth.token`, `frontend_url` → `sites.frontend`, `infernoplex_url` →
  `sites.infernoplex`, `popplio_url` → `sites.api`, `cdn_url` → `sites.cdn`,
  `proxy_url` → `meta.popplio_proxy`, `japi_key` → `japi.key`). What Popplio
  lacked lives under a new `arcadia:` section, and `servers`, `roles` and
  `channels` gained the staff-side ids.
  **Operational requirement:** the deployed `config.yaml` must gain these keys
  before startup, or config validation will fail. `./popplio --cmd genconfig`
  writes a fresh sample.
- **`PopplioStaff` is now a loopback call.** The panel proxies to `sites.api`,
  which is this same process. It still works — it is a real HTTP request through
  the normal auth path — but it is a round trip through the network stack to
  ourselves. If that becomes a problem the handler can be pointed at Popplio's
  chi router in-process.
- **One Discord session.** The staff bot attaches listeners to Popplio's existing
  disgo client rather than opening a second gateway connection. This required
  adding `IntentGuildMessages`, `IntentMessageContent` (privileged — must be
  enabled on the application) and the roles cache to Popplio's client. Upstream
  used `GatewayIntents::all()`.
- **Binary and service names** stay Popplio's (`popplio`, `popplio-staging`), not
  `bot` / `arcadia-<env>`.

**D2. The panel is a second `http.Server`, not a chi route.** Popplio's global
middleware pins `Content-Type: application/json`, caps bodies at 50 MB and applies
a 30-second timeout — all three incompatible with a protocol that returns bare
text and 204s and accepts 1 GB uploads. The panel keeps its own port (3010/3011),
its own hand-rolled mux and its own middleware chain, built on `net/http` only.

**D3. Server timeouts.** Upstream set none. Here: `ReadHeaderTimeout` 30s,
`ReadTimeout`/`WriteTimeout` 30 minutes (a 1 GB upload on a slow link must not be
killed mid-flight), `IdleTimeout` 120s.

**D4. Graceful shutdown added.** The Rust version had none. The panel server
drains, the task tickers stop via context, and `arcadia.Stop` waits up to 30s.

**D5. Startup ordering.** Upstream started the panel API from inside the Discord
`Ready` handler so the cache was warm before serving. Popplio owns the connection
and starts the panel from `main`, so the panel can accept traffic before the cache
fills. Every cache read already degrades safely: dovewing falls through to
Postgres, and role/member lookups treat an uncached guild as "not found" — which
for staff-position validation means a transient
`"Role does not exist on the staff server"` during the first seconds after boot.

**D6. Dovewing.** Popplio's `eureka/dovewing` replaces Arcadia's own three-tier
cache. Same `internal_user_cache__discord` table, same 8-hour expiry, plus a Redis
hot cache. Two visible differences: Discord's `invisible` presence maps to
`offline` (Discord only ever reports invisible users as offline to third parties,
so this is not observable in practice), and the background refresh is dovewing's
bounded one rather than a detached goroutine per lookup.

**D7. OpenAPI is hand-written.** `arcadia/panel/openapi.json`, served at
`GET /openapi` as `application/json`. utoipa has no Go equivalent and a codegen
step would need maintaining in lockstep with the union codec for no gain: the API
is two routes and one union whose shape is already pinned by round-trip tests.

**D8. `info` command build metadata.** `Rustc Version` → `Go Version`; the vergen
git sha/semver/commit-message/CPU-brand/cargo-profile fields are replaced by
`debug.ReadBuildInfo`'s VCS revision and modified flag plus `GOOS/GOARCH`. Fields
with no Go equivalent report `unknown`.

**D9. Discord command framework.** poise supplied prefix+slash parsing, checks,
cooldowns, help rendering, modals and pagination. `arcadia/bot/framework.go`
reimplements the parts the commands use. Two consequences: **per-user cooldowns
are not implemented** (poise's `user_cooldown = 3` etc. have no equivalent here),
and `register` syncs guild commands directly instead of posting poise's
registration buttons.

**D10. `chrono`-compatible timestamps.** `types.Timestamp` reproduces chrono's
`SecondsFormat::AutoSi` (0, 3, 6 or 9 fractional digits). Go's own `time.Time`
marshaller trims trailing zeroes one digit at a time and would emit `.5` where
chrono emits `.500`.

**D11. Pool size.** Upstream capped its pool at 6 connections. The port shares
Popplio's existing `pgxpool`, which uses the pgx default (`max(4, numCPU)`) — a
deliberate consequence of not standing up a second pool.

**D12. `staff_resync` position ordering.** Position id sets are sorted before
being written or logged, so SQL arrays and staff-log embeds are stable between
runs. Rust's `HashSet` iteration order was arbitrary.

---

## Testing status

| Suite | Status |
|---|---|
| `arcadia/types` — union round-trips, `Vec<u8>` as number array, chrono timestamps, null/empty encoding, `StaffMember` serialization, RPC metadata completeness | **passing** |
| `arcadia/cdnpath` — name/path validators, scope containment, granular CDN permission | **passing** |
| `arcadia/conformance` — frozen strings across panel, rpc, tasks and bot | **passing** |
| Integration tests against a seeded Postgres | **not written** |

**Known blocker:** test binaries for any package importing `popplio/state`
(`panel`, `impls`, `rpc`, `tasks`, `bot`) currently fail to *link* with
`link: github.com/bytedance/sonic/ast: invalid reference to encoding/json.unquoteBytes`.
This is pre-existing and unrelated to the port — it reproduces on a clean
checkout, and it also breaks `go build ./...` for the `popplio` binary itself on
Go 1.26.5. The cause is `bytedance/sonic v1.12.2` (pulled in via
`eureka/jsonimpl`) reaching into `encoding/json` internals that no longer exist.
Fix is to upgrade `eureka`/`sonic`, or build with an older toolchain. All arcadia
packages **compile** cleanly (`go build ./arcadia/...`) and `go vet` is clean.

That blocker is also why the security-critical CDN validators were moved into the
dependency-free `arcadia/cdnpath` package — so they are genuinely unit tested
rather than untestable in practice.

**Still to do:** §14.3 asks for integration tests against a seeded Postgres for
every SQL path, and per-operation allowed/denied permission tests. Neither is
written. Given there is no migration set in this repo (the schema is published
separately, per §13) these need a seeded database fixture first, and they are the
single highest-value thing to add next — with no compile-time query verification
to replace sqlx's, a typo'd column name will otherwise only surface in production.
