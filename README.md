# private-registry

A self-hosted Docker Registry v2 stack with **token authentication** and an **HTMX admin UI**. The Go auth service issues distribution-spec JWTs, stores users, groups, API tokens, and ACL rules in SQLite, and serves `/token` for the registry plus `/admin` for management.

## Quick start (Docker Compose)

```bash
cd deploy/compose
cp .env.example .env   # set ADMIN_PASSWORD and SESSION_SECRET
docker compose up -d --build
```

| Service  | URL |
|----------|-----|
| Auth + admin | http://127.0.0.1:8080 |
| Registry | localhost:5000 |

Health check: `curl -sf http://127.0.0.1:8080/healthz` (expect `ok`).

Log in to the registry with the bootstrap admin user:

```bash
docker login localhost:5000 -u admin -p changeme
```

Use the `ADMIN_USER` / `ADMIN_PASSWORD` from your `.env`. Create ACL rules in the admin UI before pushing images.

See [deploy/compose/README.md](deploy/compose/README.md) for smoke tests and layout details.

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_PATH` | yes | none | SQLite database file path |
| `REGISTRY_SERVICE` | yes | none | Registry service name (`aud` claim; must match registry config `service`) |
| `TOKEN_ISSUER` | yes | none | JWT issuer (`iss`; must match registry config `issuer`) |
| `TOKEN_CERT_PATH` | yes | none | Path to RSA signing certificate (PEM) |
| `TOKEN_KEY_PATH` | yes | none | Path to RSA private key (PEM) |
| `ADMIN_USER` | no* | none | Bootstrap admin username (first boot only) |
| `ADMIN_PASSWORD` | no* | none | Bootstrap admin password (first boot only) |
| `SESSION_SECRET` | yes | none | HMAC secret for admin session cookies |
| `HTTP_ADDR` | no | `:8080` | HTTP listen address |
| `TOKEN_TTL` | no | `300` | Token lifetime in seconds |

\* Required on first boot only to create the bootstrap admin user.

## ACL repository patterns

Rules use Go `path.Match` glob semantics: `*` matches within a single path segment and does **not** cross `/`.

| Pattern | Matches | Does not match |
|---------|---------|----------------|
| `danielzelisko/test-project*` | `danielzelisko/test-project`, `danielzelisko/test-project-app` | `danielzelisko/other` |
| `library/*` | `library/nginx` | `library/nginx/extra` |
| `public/*` | `public/alpine` | `public/alpine/latest` |

Create rules in the admin UI (subject = user, group, or anonymous; action = pull or push). Push implies pull.

### Anonymous pull

Add a rule with subject **anonymous**, action **pull**, and a pattern such as `public/*`. Unauthenticated `docker pull` requests are evaluated against anonymous rules only. Anonymous push is not supported.

## API token login

Create an API token for a user in the admin UI. The plaintext is shown once in the form `prt_<tokenID>_<secret>`.

Use it as the **password** with the user's username (Basic auth):

```bash
docker login localhost:5000 -u myuser -p 'prt_abc123_...'
```

The token endpoint and `docker login` accept the same credentials.

## CI and container image

Gitea Actions (`.gitea/workflows/ci.yml`) runs `go test ./...` on every push/PR, and on `main` builds and pushes:

`registry.zelisko.net/private-registry/auth:main-<unix>-<sha>`

## Kubernetes (k3s)

Raw manifests for a single-replica stack live in [deploy/k8s/](deploy/k8s/). See [deploy/k8s/README.md](deploy/k8s/README.md) for install steps. Use the published image above for `<AUTH_IMAGE>`.

**Do not scale the auth Deployment beyond 1 replica.** SQLite and the signing key files live on a single `ReadWriteOnce` PVC; multiple auth pods would corrupt the database or contend for the same key files.

## Development

```bash
go test ./...
```

## License

[MIT](LICENSE)
