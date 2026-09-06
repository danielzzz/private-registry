# Local demo

One-command walkthrough: start the stack, seed a writer and a reader, push this project's image, then pull it as the read-only user.

## Run

From the repo root:

```bash
./examples/local-demo/demo.sh
```

Requirements: Docker Compose v2 and `curl`.

```bash
cp examples/local-demo/.env.example examples/local-demo/.env
# edit ADMIN_USER, ADMIN_PASSWORD, SESSION_SECRET
./examples/local-demo/demo.sh
```

If `.env` is missing, `demo.sh` copies `.env.example` for you.

The demo stack uses **auth on port 18080** and **registry on 5000** so it does not collide with a typical app on 8080. Token realm is set in `registry-config.yml` to match.

## What it does

1. Starts `examples/local-demo/docker-compose.yml` under project `private-registry-demo`
2. Builds `Dockerfile.seed` and runs `demoseed` against the auth SQLite volume (auth is stopped briefly so SQLite is not locked)
3. Builds and pushes `localhost:5000/private-registry/auth:demo` as `writer` / `writerpass`
4. Pulls the same tag as `reader` / `readerpass`
5. Confirms that push as `reader` fails

## Demo credentials

| Where | User | Password | Access |
|-------|------|----------|--------|
| Admin UI (`:18080/admin`) | from `.env` (`ADMIN_USER`) | from `.env` (`ADMIN_PASSWORD`) | manage users/ACL |
| `docker login localhost:5000` | writer | writerpass | push/pull `private-registry/*` |
| `docker login localhost:5000` | reader | readerpass | pull `private-registry/*` |

`writer` and `reader` are registry users. They cannot sign in to the Admin UI.

## Manual testing (after the demo is up)

These steps match common checks after `./demo.sh` (or with users you create yourself).

### 1. Admin UI vs registry login

| Goal | Where |
|------|--------|
| Manage users, groups, ACL, API tokens | http://127.0.0.1:18080/admin (admin from `.env` only) |
| `docker login` / pull / push | `localhost:5000` |

Registry users must **not** use the Admin UI login page.

### 2. Create a user and grant pull (user or group)

1. Open http://127.0.0.1:18080/admin and sign in as admin.
2. **Users** → create e.g. `testuser` with a password.
3. Optional: **Groups** → create a group, add `testuser`.
4. **ACL Rules** → create a rule:
   - Subject: user `testuser`, or the group
   - Pattern: `private-registry/*`
   - Action: `pull` (or `push` if they should push; push implies pull)

Without an ACL rule, `docker login` can succeed but pull/push will be denied.

### 3. Docker login

Always pass the registry host (`localhost:5000`), not the auth port:

```bash
docker login localhost:5000 -u testuser -p 'YOUR_PASSWORD'
```

Interactive password:

```bash
docker login localhost:5000 -u testuser
```

### 4. Pull the demo image (tag matters)

The demo pushes `:demo`, not `:latest`. Omitting the tag defaults to `latest` and fails with `manifest unknown`:

```bash
# wrong (no :latest tag in this demo)
docker pull localhost:5000/private-registry/auth

# correct
docker pull localhost:5000/private-registry/auth:demo
```

List tags (after login as a user who can pull):

```bash
curl -s -u 'testuser:YOUR_PASSWORD' \
  'http://127.0.0.1:18080/token?service=registry&scope=repository:private-registry/auth:pull' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])' \
  | xargs -I{} curl -s -H "Authorization: Bearer {}" \
      http://127.0.0.1:5000/v2/private-registry/auth/tags/list
```

### 5. Typical errors

| Error | Meaning |
|-------|---------|
| Admin UI shows "Admin access required" / forbidden | Registry user tried Admin UI; use admin credentials there |
| `docker login` → `unauthorized: unauthorized` | Wrong password, or logged into Docker Hub (missing `localhost:5000`) |
| `denied: requested access to the resource is denied` | User authenticated but no matching ACL for that repo/action |
| `manifest unknown` | Tag does not exist (e.g. pulled `:latest` when only `:demo` was pushed) |

## Stop

```bash
docker compose -p private-registry-demo -f examples/local-demo/docker-compose.yml down
```

Add `-v` to delete volumes (database and signing keys).

## Notes

- The seed binary is **not** part of the production auth image; only `Dockerfile.seed` builds it.
- Re-running the demo is safe: users and ACL rules are created only if missing.
- For a minimal smoke stack without seed/push, see [deploy/compose/README.md](../../deploy/compose/README.md).
