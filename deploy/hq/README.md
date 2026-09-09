# Production deploy (hq)

Live compose, nginx, k3s pull secrets, and cutover checklist live in the **servers** repo:

- [services-docker-compose/registry/](https://git.zelisko.net/daniel/servers/src/branch/main/services-docker-compose/registry) (compose + `CUTOVER.md`)
- [k3s/manifests/registry/](https://git.zelisko.net/daniel/servers/src/branch/main/k3s/manifests/registry) (`apply-registry-secret.sh`)

This project builds `registry.zelisko.net/private-registry/auth:main-<ts>-<sha>` via `.gitea/workflows/ci.yml`. Pin that tag as `REGISTRY_AUTH_IMAGE` on hq.
