# Local demo design (2026-09-06)

## Goal

One-command local demo: start the compose stack, seed writer/reader users with ACL, build and push this project's image, then pull as the read-only user.

## Layout

- `examples/local-demo/demo.sh` — single entrypoint
- `examples/local-demo/Dockerfile.seed` — builds only `cmd/demoseed` (not shipped in the auth image)
- `examples/local-demo/README.md` — short usage
- `cmd/demoseed` — ensures demo users and ACL via `store`

Production `Dockerfile` stays auth-server-only.

## Demo users

| User | Password | ACL |
|------|----------|-----|
| writer | writerpass | `private-registry/*` push |
| reader | readerpass | `private-registry/*` pull |

Image tag: `localhost:5000/private-registry/auth:demo`

## Flow

1. Compose up via `examples/local-demo/docker-compose.yml` (`-p private-registry-demo`, auth `:18080`, registry `:5000`)
2. Wait for `http://127.0.0.1:18080/healthz`
3. Stop auth briefly, run demoseed against `auth-data` volume, start auth
4. Build, login as writer, push
5. Login as reader, pull; confirm push as reader fails
6. Print summary

Production `deploy/compose` stays the generic smoke stack; the demo has its own compose + registry-config so ports and realm stay consistent without fighting host port 8080.

## Non-goals

- No demoseed binary in the production image
- No admin UI scraping / CSRF automation
- No changes to bootstrap admin behavior

## Related product fix

Authenticated `/token` requests with no repository scopes (as used by `docker login`) now receive an empty-access JWT. Requests that ask for repository scopes and get none still return 401.
