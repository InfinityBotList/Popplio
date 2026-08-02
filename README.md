# Popplio

Popplio is the API backend for Omniplex (Previously: Infinity Bot List). It
also hosts the Arcadia staff panel API and staff Discord bot, ported into
this repository as the `arcadia/` package, and their combined background
tasks and Discord bots run in the same process as the public API.

Licensed under the AGPL-3.0. See [LICENSE](LICENSE).

## Requirements

- Go 1.23 or later
- PostgreSQL (connection string set in `config.yaml`)
- Redis (used for caching)
- A Discord bot application (for the public API's own bot) and, separately,
  a second Discord bot application for Arcadia's staff bot, since it runs
  its own gateway connection under its own identity

No platform-specific build tooling is required; Popplio is a pure Go module
with no CGO dependencies (`CGO_ENABLED=0` is used in production builds).

## Configuration

Popplio is configured via `config.yaml` at the repository root, which is
gitignored since it holds real credentials.

`config.yaml.sample` documents every available key and is regenerated
automatically from the config schema (`config/config.go`) at the start of
every startup, before `config.yaml` itself is read — so it always reflects
the current set of fields, defaults, and comments, and stays in sync on its
own whenever the schema changes. On a bare checkout with no `config.yaml`
yet, running `go run .` will still (re)write `config.yaml.sample` before it
fails on the missing `config.yaml`, so it's always safe to run first just to
get an up-to-date sample.

Copy `config.yaml.sample` to `config.yaml` and fill in the blanks. Fields
without a default in the sample (tokens, client secrets, API keys) are
required and Popplio will refuse to start without them (`validator` tags
enforce this at startup).

### Environment (staging vs production)

Many config keys are split into `staging`/`prod` pairs (see the `Differs[T]`
wrapper in `config/config.go`). Which one is actually used is controlled by
`config/current-env`, a plain text file embedded into the binary at build
time (`staging` or `prod`), not an environment variable. Edit that file and
rebuild to switch environments.

## Building and running

```
go build -o popplio .
./popplio
```

or during development:

```
go run .
```

Popplio listens on the port configured in `meta.port` (per environment).
On Linux and macOS it uses [tableflip](https://github.com/cloudflare/tableflip)
for zero-downtime restarts on `SIGHUP`; on other platforms (including
Windows) it falls back to a plain `http.ListenAndServe` with a startup
warning, since tableflip's socket handoff is not supported there. Windows
is fine for local development but is not a production target.

## Tests

```
make tests
```

Runs `go test ./...` with coverage output to `coverage.out`.

## Repository layout

This is illustrative, not exhaustive — read the actual directory before
assuming a package exists.

| Path | Contents |
|---|---|
| `main.go` | Process entrypoint: mounts every router, starts Arcadia, background loops, and the HTTP server |
| `api/` | Shared response helpers (`resp.*`) used across route handlers |
| `routes/` | Public API route handlers, one package per resource (bots, servers, teams, packs, users, votes, webhooks, ...) |
| `types/` | Request/response types shared across routes |
| `state/` | Process-wide globals (Postgres pool, Redis client, Discord session, logger, parsed config) initialised once at startup |
| `config/` | Configuration schema and the embedded `current-env` file |
| `teams/` | Team/entity permission resolution (built on `kittycat`) |
| `webhooks/` | Outbound webhook delivery |
| `notifications/` | Push notifications and vote reminders |
| `votes/`, `shop/`, `payments/` (under `routes/`) | Voting, shop/coupons, PayPal/Stripe payment handling |
| `arcadia/` | The staff panel API and staff Discord bot, ported from the standalone Rust Arcadia service. See `arcadia/CONFORMANCE.md` for what was intentionally reproduced byte-for-byte versus fixed during the port |
| `cmd/kitehelper/` | Standalone maintenance CLI (DB migration helpers, image proxy, table validation) — separate `go.mod`, built independently |
| `exp/` | One-off SQL scripts for schema changes applied manually against the live database, not a migration framework |
| `data/docs.html` | Template served at `/docs` |

## API documentation

Live OpenAPI docs are served by the running instance at `/docs`
(production: https://spider.omniplex.gg/docs).

## Working with Discord users

Always fetch Discord user data through `dovewing.GetUser`, not a raw
Discord API call it transparently handles gateway cache, Redis, and
in-memory caching, and every other part of the codebase assumes user data
went through it.
