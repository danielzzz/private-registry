# Compose stack: auth + registry2

Local smoke stack for the private-registry auth app and stock Docker Registry v2 with token authentication.

## Prerequisites

- Docker with Compose v2
- Copy `.env.example` to `.env` and set `ADMIN_PASSWORD` and `SESSION_SECRET`

```bash
cp .env.example .env
```

## Start

From this directory:

```bash
docker compose up -d --build
```

Auth listens on `http://127.0.0.1:8080`, registry on `localhost:5000`.

## Smoke test

1. Wait for auth health:

   ```bash
   curl -sf http://127.0.0.1:8080/healthz
   ```

   Expected: `ok`

2. Registry should challenge unauthenticated clients:

   ```bash
   curl -si http://127.0.0.1:5000/v2/
   ```

   Expected: `401` with `Www-Authenticate` pointing at `http://127.0.0.1:8080/token`

3. Log in from the host (realm must be host-reachable; `registry-config.yml` uses `http://127.0.0.1:8080/token`):

   ```bash
   docker login localhost:5000 -u admin -p changeme
   ```

   Use the `ADMIN_USER` / `ADMIN_PASSWORD` from your `.env`.

4. Optional push/pull smoke (after login):

   ```bash
   docker pull alpine:3.20
   docker tag alpine:3.20 localhost:5000/smoke/alpine:test
   docker push localhost:5000/smoke/alpine:test
   ```

   Push requires ACL rules for the target repository (create via the admin UI at `http://127.0.0.1:8080/admin`).

## Layout

| Path | Purpose |
|------|---------|
| `docker-compose.yml` | `auth` and `registry` services |
| `registry-config.yml` | Registry token auth (realm, issuer, root cert) |
| `../../Dockerfile` | Multi-stage build for the Go auth server |
| `.env.example` | Required env vars |

Shared volume `certs`: auth generates `token.crt` / `token.key` on first boot; registry reads the cert as `rootcertbundle`.

## Notes

- `registry-config.yml` `issuer` and `service` must match `TOKEN_ISSUER` and `REGISTRY_SERVICE` in `.env`.
- For `docker login` from the host, the token **realm** URL must resolve on the host (not `http://auth:8080/token`). If you change `AUTH_HOST_PORT`, update `realm` in `registry-config.yml` to match (e.g. `http://127.0.0.1:18080/token`).
- Do not scale auth to multiple replicas against one SQLite file.

## Stop

```bash
docker compose down
```

Add `-v` to remove volumes (database and signing keys).
