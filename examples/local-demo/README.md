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

## Stop

```bash
docker compose -p private-registry-demo -f examples/local-demo/docker-compose.yml down
```

Add `-v` to delete volumes (database and signing keys).

## Notes

- The seed binary is **not** part of the production auth image; only `Dockerfile.seed` builds it.
- Re-running the demo is safe: users and ACL rules are created only if missing.
- For a minimal smoke stack without seed/push, see [deploy/compose/README.md](../../deploy/compose/README.md).
