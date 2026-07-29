# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Nothing has been tagged or released yet — everything below falls under the
initial `1.0.0` version.

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

### Security

- Internal error details (raw Go `error.Error()` text) are no longer
  included in several API error responses, including session
  authentication and a number of other endpoints the underlying error is
  still logged server-side in full, but clients now receive a generic
  message instead of internal implementation details.
