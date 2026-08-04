# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - Unreleased

### Added

- `current-env` now also accepts `beta`, a fourth environment alongside
  `staging`/`prod`/`dev`. Every `Differs[T]` config key gains an optional
  `beta` value (`config.Differs[T].Beta`), consulted only when `current-env`
  is `beta` and falling back to `staging` when unset — same mechanism as
  `dev`'s override, but without `dev`'s relaxed Staging/Prod requirement:
  `beta` is validated exactly like `staging`/`prod` (`ValidateDiffers`),
  since it's a real running deployment rather than a personal machine. In
  practice this means most config (DB, tokens, etc.) can stay shared with
  staging, and only keys that genuinely differ per deployment — like
  `sites.frontend` — need an explicit `beta:` value.
- `bgtasks` package: a new home for Popplio's own periodic background jobs,
  separate from `arcadia/tasks` (the staff bot's jobs, which only run when
  Arcadia is configured) so core platform features don't depend on staff
  tooling being set up. First job: `bot_uptime_check`, which periodically
  records whether every listed bot is currently online in the main server
  into `bots.uptime`/`total_uptime`/`uptime_last_checked`. These columns
  have existed since the Rust port but were never actually written to —
  Arcadia's old uptime checker (`src/tasks/__toberewritten/uptime.rs`)
  didn't even compile against the serenity version it was last touched
  against, and was explicitly never ported (see `arcadia/CONFORMANCE.md`).
  Reads presence straight from Popplio's own gateway cache (it already
  requests the Presence intent) rather than Infernoplex, which deliberately
  never requests it.
- `servers.avatar`: servers previously had no icon anywhere (index listing,
  detail page, or the staff panel's server search all showed a blank/
  initials fallback) — the old cache-server subsystem used to synthesize
  this from its own CDN cache, and nothing replaced it after that was
  retired (`exp/remove_cache_servers.sql`). Populated once at Add Server
  time from the invite resolution already done there, and kept fresh
  afterward by Infernoplex's `serversync` task, which now also syncs every
  listed server's icon (not just opted-in ones' emojis/stickers) from its
  gateway cache. Requires the new `servers.avatar` column
  (`exp/add_servers_avatar.sql`, needs to be applied manually against the
  database like other `exp/` scripts).
- Webhooks gained a new `hmac_auth` mode (`hmac_auth` on
  `POST`/`PATCH .../webhooks`): the payload is sent as plain JSON with an
  `X-Webhook-Signature: sha256=<hex hmac>` header, the same shape GitHub and
  Stripe webhooks already use. It's now the recommended mode for new
  webhooks the previous default ("splashtail": AES-GCM encrypted body,
  nonce-chained double HMAC across two headers) required implementing
  decryption just to verify a delivery, not just a signature check.
  Existing webhooks are unaffected: `hmac_auth` defaults to off and the
  splashtail/`simple_auth` protocols are unchanged and fully supported 
  this only adds a third option, it doesn't remove or alter the other two.
  Requires the new `webhooks.hmac_auth` column
  (`exp/webhookhmacauth.sql`, needs to be applied manually against the
  database like other `exp/` scripts).
- `current-env` now also accepts `dev`, a third environment alongside
  `staging`/`prod`. Every `Differs[T]` config key (`config/config.go`) gains
  an optional `dev` value, only consulted when `current-env` is `dev`, and
  only used if actually set — an unset `dev` value falls back to `staging`,
  so no existing `config.yaml` needs to change. Lets a local checkout run
  against things like a personal Discord bot application
  (`discord_auth.token`, `arcadia.token`) without touching the real staging
  config. `discord_auth.token` (Popplio's own bot token) is now itself a
  `Differs[string]` rather than a single flat value, so it can differ across
  environments the same way Arcadia's staff bot token already could.
  Anything gated to "real production" (Paypal live vs sandbox API base,
  Arcadia's background tasks, the staff bot's guild-member-join
  announcements, the staging-sensitive-permission gate) now treats `dev` the
  same as `staging` rather than falling through to production behavior.

- `PUT /servers` add a server to the list directly from a Discord invite
  link. Resolves the guild via the invite (the tracking bot does not need to
  already be in the server), rejects duplicates and blacklisted vanities,
  and auto-creates an owning team the same way bot submission already does
  (or attaches to an existing team the submitter has `bot.add`-equivalent
  permission on).
- Packs can now include servers alongside bots: a `servers` column,
  resolution into full `IndexServer` objects, and matching validation on
  both `POST /packs` (create) and `PATCH /packs/{url}` (edit). A pack must
  contain at least one bot or server between the two fields.
- Bots can self-report presence (`online`/`idle`/`dnd`/`offline`) via
  `POST /bots/stats`, alongside the existing server/shard/user stats. The
  reported value is folded into the resolved `user.status` returned
  everywhere a bot's info appears, since most bots don't share a guild with
  the tracking bot for a real gateway presence to be read from.
- Bots with no explicit self-reported status but a real track record of
  posting stats (a nonzero server count from a stats post within the last
  24 hours) are now shown as `online` rather than falling back to
  dovewing's almost-always-offline gateway-derived status.
- `GET /servers/meta?invite=...` resolves a Discord invite to a preview of
  the server it points to (name, icon, member counts, and whether it's
  already listed) without adding anything — lets a client show what's about
  to be submitted before Add Server is actually called. Shares its invite
  resolution logic with `PUT /servers` via a new `ResolveInvite` helper.
- Servers can opt in to showing their custom emojis and stickers on their
  listing page via a new `show_emojis` setting (`PATCH /servers/{id}/settings`).
  `GET /servers/{id}` now includes `emojis`/`stickers`/`emojis_synced_at`,
  always empty unless the owner has opted in. The actual snapshot is synced
  periodically by the tracking bot (Infernoplex), not fetched live per
  request, and requires the bot to currently be a member of the server —
  Popplio itself never talks to Discord for this.
- `GET /servers/meta` now also reports `bot_present`/`bot_invite_url` by
  asking Infernoplex's Sorbet API whether the tracking bot is currently a
  member of the resolved guild, via a new `CheckBotGuildPresence` helper.
  Best-effort: any failure to reach Infernoplex is treated as "not present"
  rather than failing the request.

### Changed

- `meta.popplio_proxy` now defaults to `https://gateway.nodebyte.host/proxy/discord`
  (the shared parent-company gateway), replacing the old local
  `http://127.0.0.1:3219` twilight-http-proxy convention. Both Popplio's own
  bot client (`state.Setup`) and Arcadia's separate staff bot
  (`arcadia/dclient`) now route their REST traffic through it via
  `rest.WithURL`/`rest.WithHTTPClient` (`state.ProxyRestOpts`). Since that
  gateway authenticates every request with its own shared bot credential by
  default, each client sends its own token via an `X-Upstream-Authorization`
  header instead, which the gateway forwards as the real `Authorization`
  header sent to Discord — so Popplio and Arcadia's staff bot each keep
  their own distinct bot identity rather than both authenticating as
  whichever bot the gateway holds.
- `EntityGetVoteCount` (used by nearly every bot/server/team/user/pack
  detail and list endpoint) now counts up- and down-votes in a single query
  with `FILTER`, instead of two separate `COUNT(*)` round trips.
- Bot/server index resolution (`ResolveIndexBot`/`ResolveIndexServer`,
  called by `GET /bots/@all`, `GET /servers/@all`, search, random, the bots
  index, packs, team entities, and user profiles) now resolves every row in
  a page concurrently via `errgroup` instead of one row at a time — each
  row's dovewing/vanity/vote lookups are independent, so a page of results
  no longer pays for them sequentially.
- `GET /list/current-status` now issues both the Instatus and UptimeRobot
  requests with the request's own context and a bounded client timeout,
  instead of an unbounded `http.Get`/`http.NewRequest` that could hang the
  handler indefinitely if the upstream stalled.
- Deduplicated the `page` query-parameter parsing copy-pasted across nine
  endpoints (each with a slightly different error response for the same
  invalid-page case) into a shared `pagination.Parse` helper.
- `DELETE /users/{uid}/packs/{id}` and `PATCH /users/{uid}/packs/{id}` each
  folded two sequential "does the pack exist" / "who owns it" queries into
  one.
- The generic error bodies returned when a failure carries no specific
  message of its own (`constants/constants.go` — 404s, 400s, 403s, 401s,
  500s, 405s, and missing-body errors) were all a "Slow down, bucko!" joke
  string. Replaced with plain, professional messages that actually describe
  the failure.

### Fixed

- `PUT /servers` and `PUT /bots` still wrote the legacy wildcard string
  `global.*` into a new team's `team_members.flags` when creating the
  owner's membership, instead of the flat model's `owner` permission
  (`perms.EntityOwner`). `exp/rewrite/flatperms.sql` converts this
  correctly for existing rows, but every server/bot added *after* running
  that migration created a team whose owner held a permission string the
  flat permission checker doesn't recognize as anything — silently locking
  them out of managing their own new listing (`edit_servers`/`edit_bots`
  checks fail, since `global.*` isn't `owner` and isn't a declared
  permission either). `arcadia/tasks/cleaners.go`'s `TeamCleaner` task had
  the same bug in both directions: it looked for orphaned-of-owner teams by
  querying `flags @> ARRAY['global.*']` (which will now never match
  anything, since the migration already converted every existing row) and
  wrote `global.*` back when promoting a replacement owner. All three now
  use `perms.EntityOwner`.
- Every mod-log notification embed (`PUT /servers`, `PUT /bots`,
  `DELETE /bots/{id}`, and the two `PATCH .../settings` endpoints below)
  built its link back to the site with `Sites.Frontend.Production()`,
  forcing the production URL regardless of which environment the action
  actually happened in. On staging/beta, this meant either a broken link
  (if `sites.frontend.prod` wasn't configured on that box at all — its
  `Production()` has no fallback, so an unset value silently becomes an
  invalid relative-path embed URL and Discord rejects the whole message
  with `50035`) or a link to an entity that only exists in a different
  environment's database. Switched to `.Parse()`, which resolves against
  whichever environment is actually running.
- `PATCH /servers/{id}/settings` and `PATCH /bots/{id}/settings` returned a
  500 whenever their mod-log notification embed failed to send — including
  the guaranteed case for servers, which built its embed with
  `Thumbnail: &discord.EmbedResource{}` (a present-but-empty resource,
  which Discord's API rejects outright with `50035: Invalid Form Body`
  rather than treating as "no thumbnail"). The underlying update had
  already succeeded in both cases — the error message even said so — so a
  caller retrying on this 500 risked double-submitting. The thumbnail is
  now omitted when there's no avatar instead of sent empty, servers' embed
  now uses the real `servers.avatar` value instead of nothing, and a
  failure to post the notification is logged rather than failing the
  request, matching the existing pattern in `PUT /servers`.
- `PUT /servers` wrote every field into the wrong column: `createServerArgs`'s
  hand-written value order didn't match `types.CreateServer`'s field
  declaration order, which is what `db.GetCols`/the generated column list
  actually follow. Values were bound to columns purely by position, so e.g.
  `server_id` was written into `invite`, `name` into `short`, and
  `extra_links` (a `[]Link`) into `tags` (a `text[]`) — the last of which is
  what surfaced as a `cannot find encode plan` error, since a `[]Link` can't
  encode into a `text[]` column. `createServerArgs` now lists values in the
  same order as the struct, with a comment on both explaining they must stay
  in sync (the existing length check on `createServerColsArr`/`serverArgs`
  only ever caught the two lists having different lengths, not entries being
  out of order relative to each other).
- `servers.extra_links` was still `text[]` in the deployed database while
  the application code (and `data/seed-ci.json`'s own schema conformance
  check) has expected `jsonb` for a while, matching `bots`/`teams`/`users`'
  `extra_links` columns. Migrated via `exp/fix_servers_extra_links_type.sql`
  (needs to be applied manually against the database like other `exp/`
  scripts, same as `exp/rewrite/*.sql`). Independent of the column-ordering
  bug above, but was masking it: the ordering bug meant `extra_links`
  received whatever value happened to land at that position, which could
  vary by field type in ways that made this fix look sufficient on its own.
- Startup logged `error while setting presence err="no gateway configured"`
  on every run: `Discord.SetPresence` was called right after
  `Discord.OpenShardManager` returned, but that only means the shards
  started connecting, not that the gateway session is actually usable yet
  (that's only confirmed later, asynchronously, via the `OnGuildsReady`
  event). `SetPresence` now runs from inside the `OnGuildsReady` handler
  instead, once shards are confirmed ready.
- `current-env: dev` still required a real `staging` and `prod` value for
  every `Differs[T]` config key, and for every Arcadia staff-server
  channel/role/server ID, defeating the point of `dev`: a local checkout
  needed a fully populated staging/prod config, including Arcadia secrets,
  just to start. `Differs[T]`'s requirement is now environment-aware (`dev`
  only needs `dev` or a `staging` fallback to resolve; `staging`/`prod`
  keep the original both-required behavior), and Arcadia's staff-server
  fields use a new `requirednotdev` validator so they're only required
  outside of `dev`.
- `GET /servers/meta`'s route registration was missing the `ExtData` entry
  `BaseSanityCheck` requires whenever a route declares `Auth`, so the
  server panicked at startup ("Base sanity check failed: permissionCheck
  not found in route.ExtData") before it could serve a single request.
- `PATCH .../webhooks/{id}` silently ignored `simple_auth` in the request
  body — the `UPDATE` statement never included that column, so a webhook's
  auth mode could only ever be set at creation, never changed afterward.
- `GET /list/current-status`'s Redis cache never actually worked: it passed
  a raw `map[string]any` to `Set`, which go-redis cannot serialize (returns
  `"redis: can't marshal map[string]interface{}"`) — an error that was
  never checked, so every request silently round-tripped to Instatus or
  UptimeRobot instead of using the 3-minute cache the code's own comment
  says exists. Now JSON-marshals before `Set` and unmarshals back into the
  same shape on a hit, so the response is identical whether it came from
  cache or not (previously a hit, had it ever occurred, would have returned
  a raw string instead of the status object a miss returns).
- `add_review`/`edit_review`/`remove_review` each called
  `state.Redis.Del(ctx, "rv-"+targetId+"-"+targetType)` on every mutation,
  invalidating a cache key that is never `Set` or `Get` anywhere in the
  codebase — three no-op Redis round trips per review change. Removed.
- The OAuth authorization-code replay check (`create_oauth2_login`) used a
  separate `Exists` followed by `Set` to mark a code used, which is not
  atomic: two concurrent requests carrying the same code could both pass
  `Exists` before either called `Set`, letting the same code be redeemed
  twice — exactly the race the code's own comment says it closes, but
  didn't. Now uses `SetNX`, which checks and marks the code used in one
  atomic round trip.
- Several route handlers (`delete_pack`, `patch_pack`, `current_status`)
  returned a bare `uapi.DefaultResponse(http.StatusInternalServerError)` on
  DB/upstream failures without logging anything, so some production 500s
  were invisible in the logs. They now go through `resp.Err`, which is what
  the shared `api/resp` package exists for.
- A background goroutine filtering empty entries out of the Stripe webhook
  IP allowlist mutated the slice while ranging over its original indices, a
  classic Go bug that silently skips the element shifted into a just-removed
  slot — consecutive empty lines in Stripe's IP list could leave stale
  entries in a security-relevant allowlist. Now builds a filtered copy
  instead of mutating in place.
- Startup panicked the entire process on a transient Stripe API/network
  failure (deleting existing webhooks, creating the new one, or fetching its
  IP allowlist) instead of disabling Stripe webhook support and continuing,
  unlike the equivalent Paypal setup a few lines above it, which already
  degraded gracefully.
- `webhooks/sender` used `panic()` as its input-validation strategy for a
  handful of preconditions, including from inside an unrecovered goroutine
  (the randomized "send a bad webhook to test auth" path) — a single
  malformed webhook payload reaching that path could crash the whole
  process rather than just fail one webhook delivery. Preconditions now
  return errors instead.
- Several fire-and-forget goroutines spawned from request handlers (bot/server
  detail-page analytics, review garbage collection, Stripe perk delivery,
  vote logging and webhook dispatch) had no `recover()`, so a panic in any of
  them would take down the whole process instead of just that background
  task. Two of them (`create_user_entity_vote`'s webhook-send goroutine and
  the Stripe perk-delivery goroutine) also wrote to an `err` variable shared
  with their enclosing handler, a data race; both now use a goroutine-local
  variable.
- `data/seed-ci.json`, the schema snapshot the `db_fields_check.py` CI test
  checks Go struct `db` tags against, had fallen out of sync with several
  recently added columns (`bots.self_status`, `packs.servers`,
  `servers.show_emojis`/`emojis`/`stickers`/`emojis_synced_at`), breaking the
  test build. Also found and excluded `bots.cache_server_uninvitable`, a
  real DB column with no corresponding Go struct field anywhere in the
  codebase, via the existing `ignore_fields` convention.
- Team votes were never resolved when a team was embedded inside a user's
  profile response (`GET /users/{id}`) — every embedded team silently
  reported 0 votes regardless of its real count.
- `GET /users/{id}` never requested team member data (`team_member`) when
  resolving a user's teams, only `bot` and `server` — so any client relying
  on that response to determine a user's permissions on a team-owned entity
  (e.g. "can I edit this bot?") always saw an empty permission set.
- The tracking bot's Discord presence update was hard-gated to the
  production environment only, and silently swallowed any error from
  actually setting it (the failure check was reading a stale variable from
  an earlier, unrelated call rather than the real result) — presence now
  updates in every environment and logs real failures instead of going
  silent.
- Nil slices (a user's teams, a team's `bots`/`servers`/`members`, etc.) now
  consistently marshal as `[]` instead of `null` across list/user/team
  endpoints, preventing frontend crashes on `.length`/`.map` against a
  response that has no data yet.
- `GET /teams/{id}` had the same nil-slice gap on `tags`/`extra_links`
  directly (as opposed to the application-resolved slices above, which were
  already covered) — a team with no tags or links set crashed the frontend
  team page outright rather than just rendering emptily.
- `webhooks/sender` failed to build at all: an in-progress refactor had
  extracted `Secret`, `webhookSendState`, and several helper methods
  (`resolveTarget`, `buildRequest`, `notify`, `markFailed`, `logFields`)
  into new `request.go`/`sendstate.go` files, but `sender.go` still carried
  the old duplicate declarations and its own inline copies of the same
  logic. `sender.go` now uses the extracted helpers instead of duplicating
  them.

### Security

- Internal error details (raw Go `error.Error()` text) are no longer
  included in several API error responses, including session
  authentication and a number of other endpoints the underlying error is
  still logged server-side in full, but clients now receive a generic
  message instead of internal implementation details.
