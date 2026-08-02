# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - Unreleased

### Added

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

### Fixed

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
