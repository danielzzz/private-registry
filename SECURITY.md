# Security

## Report a vulnerability

Use [GitHub Security Advisories](https://github.com/danielzzz/private-registry/security/advisories/new) for this repository.

Do not open a public issue for security bugs until there is a fix or an agreed disclosure date.

## What this project assumes

- Auth and registry sit behind TLS (reverse proxy or Ingress). The app itself speaks HTTP.
- You set a strong `ADMIN_PASSWORD` and a long random `SESSION_SECRET`. Example values like `changeme` are for local demos only.
- Auth credentials (SQLite file or MySQL DSN) and RSA signing keys under your data volume are treated as secrets. Losing the signing key invalidates existing registry tokens; leaking it lets an attacker mint tokens.

## Known limits

- Admin session cookies are `HttpOnly` and `SameSite=Lax`, but not `Secure`. Rely on HTTPS at the proxy, or terminate TLS only on trusted networks.
- Login and `/token` rate limits are in-memory per process. They reset on restart and do not coordinate across hosts.
- The admin UI loads Tailwind from a CDN. Pin or vendor the CSS if that supply chain is unacceptable for you.
- Auth must stay at one replica while signing keys live on a single PVC.

## Production checklist

1. TLS in front of auth and registry
2. Unique bootstrap admin password and session secret
3. Persistent volume for `/data` when using auto-generated keys and/or SQLite; or mount signing PEMs from a Secret / bind mount (see README “Signing certificates”)
4. Backup signing keys (and SQLite if used) on a schedule you trust
5. ACL rules that default-deny; grant pull/push explicitly
6. Prefer API tokens for CI over sharing the admin password
7. One auth replica unless every replica shares the same signing key
