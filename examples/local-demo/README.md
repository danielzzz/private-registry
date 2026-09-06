# Local demo

One-command walkthrough: start the stack, seed a writer and a reader, push this project's image, then pull it as the read-only user.

## Run

From the repo root:

```bash
./examples/local-demo/demo.sh
```

Requirements: Docker Compose v2 and `curl`. If `deploy/compose/.env` is missing, the script copies `.env.example`.

The demo stack uses **auth on port 18080** and **registry on 5000** so it does not collide with a typical app on 8080. Token realm is set in `registry-config.yml` to match.

## What it does

1. Starts `examples/local-demo/docker-compose.yml` under project `private-registry-demo`
2. Builds `Dockerfile.seed` and runs `demoseed` against the auth SQLite volume (auth is stopped briefly so SQLite is not locked)
3. Builds and pushes `localhost:5000/private-registry/auth:demo` as `writer` / `writerpass`
4. Pulls the same tag as `reader` / `readerpass`
5. Confirms that push as `reader` fails

## Demo credentials

| User | Password | Access |
|------|----------|--------|
| writer | writerpass | push/pull `private-registry/*` |
| reader | readerpass | pull `private-registry/*` |

Admin remains the bootstrap user from `deploy/compose/.env`.

## Stop

```bash
docker compose -p private-registry-demo -f examples/local-demo/docker-compose.yml down
```

Add `-v` to delete volumes (database and signing keys).

## Notes

- The seed binary is **not** part of the production auth image; only `Dockerfile.seed` builds it.
- Re-running the demo is safe: users and ACL rules are created only if missing.
- For a minimal smoke stack without seed/push, see [deploy/compose/README.md](../../deploy/compose/README.md).
