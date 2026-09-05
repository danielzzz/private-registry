# Registry auth service design

MIT-licensed Go service that sits beside Docker Registry 2 and provides token auth, users/groups, wildcard ACLs, and a small HTMX admin UI.

Target users: solo operators and small teams. Deploy with Compose or a single-replica k3s chart.

## Goals

- `docker login` / `pull` / `push` work with the normal Docker (and OCI) client flow
- Local users plus API tokens for CI
- Groups and per-user ACL rules with repository globs (e.g. `danielzelisko/test-project*`)
- Anonymous pull for selected patterns
- Admin UI to manage users, tokens, groups, and rules (no image catalog in v1)
- SQLite only, one app replica
- TDD: failing tests first, then implementation

## Non-goals (v1)

- Full reverse proxy in front of registry2
- Postgres / multi-replica HA
- SSO (OIDC/LDAP)
- Catalog browser or delete-tag/delete-repo UI
- Self-service signup
- Anonymous push
- Regex ACL patterns (globs only)

## Architecture

Two processes:

1. **registry2** (upstream image): content storage. Configured for token auth.
2. **auth app** (this project): issues JWTs, stores ACL data, serves admin UI.

```text
docker client
  ├─ pull/push ──► registry2 ──401 + realm──► client
  │                      ▲
  │                      └── Bearer JWT (registry verifies with shared cert)
  │
  └─ get token ──► auth app /token
                      │
                      ├─ Basic auth (password or API token) or anonymous
                      ├─ evaluate ACL
                      └─ signed JWT with allowed scopes
```

Admins use the same app over HTTPS for the UI (cookie session). CI uses `docker login` with username + API token as password.

Persistent data (volume / PVC):

- SQLite database
- JWT signing key and certificate (registry trusts the cert)

## Auth model

### Registry token endpoint

Implements the Docker Registry v2 Bearer token flow.

- Authenticate with HTTP Basic: username + password, or username + API token
- Or unauthenticated, for anonymous pull evaluation only
- Request includes `service` and `scope` query params (`repository:<name>:<action>`)
- Response is a JWT the registry accepts (`iss`, `aud`/`access`, expiry, signed with the configured key)

If the caller is authenticated (or anonymous) but **no** requested scope is allowed, respond with **401**. Do not issue an empty-access token.

### Admin sessions

Separate from registry tokens. Cookie-based session after UI login. Only users with `admin` can use mutating admin routes.

### Bootstrap

On empty database, create one admin from `ADMIN_USER` / `ADMIN_PASSWORD`. If either is unset when the DB has no users, the process exits with an error. No web-based first-run wizard in v1.

## Data model

### User

- username (unique)
- password hash (argon2id or bcrypt)
- active
- admin

### API token

- belongs to user
- name/label
- secret hash
- optional expiry
- active
- plaintext secret shown once at creation

### Group

- name
- members (users)

### ACL rule

- subject: user id, group id, or anonymous
- repository pattern (glob on repository path only, not tags)
- action: `pull` or `push` (`push` implies `pull`)

### Pattern matching

- v1 globs with `*` (e.g. `library/*`, `danielzelisko/test-project*`)
- Match against the repository name in the scope, not the tag
- No full regex in v1

### ACL evaluation

1. Resolve subjects: `anonymous`, and if authenticated the user plus their groups
2. For each requested scope, allow if any rule for those subjects matches the repository pattern and grants the action
3. Union of allowed scopes goes into the token
4. If that set is empty → 401

## Admin UI

Stack: Go `html/template`, HTMX, Tailwind. v1 may use the Tailwind CDN to avoid a frontend build step; switch to a built CSS file later if needed.

Pages (admin only after login):

- Dashboard (counts + short registry config hint)
- Users (create, disable, reset password, toggle admin)
- API tokens (create/revoke for a user)
- Groups (create, membership)
- ACL rules (CRUD; subject = user | group | anonymous)
- Login / logout

No catalog, no self-service account area beyond login.

## HTTP surface

| Path | Purpose |
|------|---------|
| `GET /token` | Registry token endpoint |
| `GET /healthz` | Liveness/readiness |
| `/login`, `/logout`, `/admin/...` | Admin UI |

CSRF protection on mutating admin form posts. Light in-memory rate limit on `/token` and login.

## Configuration (env)

| Variable | Purpose |
|----------|---------|
| `DATABASE_PATH` | SQLite file path |
| `HTTP_ADDR` | Listen address |
| `REGISTRY_SERVICE` | Token service name; must match registry config |
| `TOKEN_ISSUER` | JWT issuer |
| `TOKEN_CERT_PATH` / `TOKEN_KEY_PATH` | Signing material; generate on first boot if missing (dev-friendly) |
| `ADMIN_USER` / `ADMIN_PASSWORD` | Bootstrap admin |
| `SESSION_SECRET` | Admin cookie signing |

TLS terminates at Ingress / Caddy / Traefik in normal deploys. App speaks HTTP inside the cluster or Compose network.

## Deployment

### Compose

- Services: `registry` + `auth`
- Shared or dedicated volumes for DB and keys
- Documented registry config pointing `auth.token.realm` at the auth service
- Example suitable for local smoke tests with real `docker` CLI

### k3s

- One replica Deployment
- PVC for SQLite + keys
- Service + Ingress for registry host and auth realm URL (same host/path split or separate host)
- Helm chart or manifests with values for domain, storage class, bootstrap secrets

Document clearly: **do not run multiple replicas against one SQLite file**.

## Project layout

```text
cmd/server/          main
internal/auth/       credential verify, JWT issue
internal/acl/        glob + rule evaluation
internal/store/      SQLite
internal/web/        admin routes, templates, static
deploy/compose/      Compose + registry config examples
deploy/k8s/          manifests or Helm chart
docs/                specs, user docs later
```

## Development process: TDD

All feature work follows test-driven development:

1. Write a failing test that specifies the behavior
2. Run it and confirm it fails for the right reason
3. Write the smallest implementation that passes
4. Refactor while keeping tests green

Priority order for tests:

1. Unit: repository glob matching
2. Unit: ACL evaluation matrix (user, group, anonymous, pull/push, wildcards, push-implies-pull, deny-by-omission)
3. Unit: JWT claims (issuer, service, access entries, expiry)
4. Integration: SQLite store + `/token` + admin CRUD over `httptest`
5. Manual/Compose smoke documented for real `docker login/pull/push`

Do not add production code for a behavior until a failing test exists for it.

## License

MIT.

## Open decisions for the implementation plan

Narrow items to lock while planning:

- Exact password KDF (argon2id preferred)
- Helm chart vs raw k8s manifests for the first k3s package
- Default token lifetime
- Default JWT expiry clock skew tolerance if any
