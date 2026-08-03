# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - Unreleased

### Added

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

### Fixed

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
