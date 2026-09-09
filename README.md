# private-registry

Self-hosted Docker Registry v2 with token authentication and an HTMX admin UI. The Go auth service issues distribution-spec JWTs, stores users, groups, API tokens, and ACL rules in SQLite, and serves `/token` for the registry plus `/admin` for management.

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

Log in to the registry with the bootstrap admin user from your `.env` (examples use `admin` / `changeme` for local demos only):

```bash
docker login localhost:5000 -u admin -p changeme
```

Create ACL rules in the admin UI before pushing images.

See [deploy/compose/README.md](deploy/compose/README.md) for smoke tests and layout details.

### Full local demo (seed users, push, pull)

```bash
./examples/local-demo/demo.sh
```

Auth on `http://127.0.0.1:18080`, registry on `localhost:5000`. Manual steps: [examples/local-demo/README.md](examples/local-demo/README.md).

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
| `myorg/app*` | `myorg/app`, `myorg/app-api` | `myorg/other` |
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

## Production notes

Local Compose is HTTP and uses demo defaults. For a real deployment:

1. Terminate TLS in front of auth and registry (reverse proxy or Ingress).
2. Set a strong `ADMIN_PASSWORD` and a long random `SESSION_SECRET`. Never keep `changeme`.
3. Persist the auth data volume (SQLite + `token.crt` / `token.key`). Back it up. Losing the signing key breaks existing tokens.
4. Keep the auth Deployment at **one replica** when using this SQLite layout.
5. Build and push your own image from the repo `Dockerfile`; pin that tag in k8s or Compose.

See [SECURITY.md](SECURITY.md) for reporting and known limits.

## Build the auth image

```bash
docker build -t your-registry.example/private-registry/auth:TAG .
docker push your-registry.example/private-registry/auth:TAG
```

## CI

GitHub Actions (`.github/workflows/ci.yml`) runs `go test ./...` on push and pull requests.

## Kubernetes (k3s)

Raw manifests for a single-replica stack live in [deploy/k8s/](deploy/k8s/). See [deploy/k8s/README.md](deploy/k8s/README.md). Set the auth container image to the tag you built above.

**Do not scale the auth Deployment beyond 1 replica.** SQLite and the signing key files live on a single `ReadWriteOnce` PVC; multiple auth pods would corrupt the database or contend for the same key files.

## Development

```bash
go test ./...
```

## License

[MIT](LICENSE)
