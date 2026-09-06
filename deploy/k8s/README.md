# k3s manifests (auth + registry2)

Raw Kubernetes manifests for a single-replica private registry stack on k3s.

## Prerequisites

- k3s (or any Kubernetes cluster with an Ingress controller)
- Replace placeholders before applying:
  - `<AUTH_IMAGE>` in `deployment-auth.yaml`
  - `<STORAGE_CLASS>` in `pvc.yaml` (k3s default: `local-path`)
  - `registry.example.com` and `auth.example.com` in `ingress.yaml`
  - `realm` URL in `deployment-registry.yaml` ConfigMap (must match the auth Ingress host)

## Install

```bash
# 1. Create secrets (do not commit secret.yaml)
cp secret.example.yaml secret.yaml
# edit secret.yaml: ADMIN_USER, ADMIN_PASSWORD, SESSION_SECRET

kubectl apply -f namespace.yaml
kubectl apply -f pvc.yaml
kubectl apply -f secret.yaml
kubectl apply -f deployment-auth.yaml
kubectl apply -f service-auth.yaml
kubectl apply -f deployment-registry.yaml
kubectl apply -f service-registry.yaml
kubectl apply -f ingress.yaml
```

Or apply all manifests except the example secret:

```bash
kubectl apply -f deploy/k8s/namespace.yaml \
  -f deploy/k8s/pvc.yaml \
  -f deploy/k8s/secret.yaml \
  -f deploy/k8s/deployment-auth.yaml \
  -f deploy/k8s/service-auth.yaml \
  -f deploy/k8s/deployment-registry.yaml \
  -f deploy/k8s/service-registry.yaml \
  -f deploy/k8s/ingress.yaml
```

## Single replica only

**Do not scale the auth Deployment beyond 1 replica.** SQLite and the signing key files live on a single `ReadWriteOnce` PVC. Multiple auth pods would corrupt the database or fight over the same files.

Registry image blobs use `emptyDir` in these manifests (ephemeral). For production, add a separate PVC for `/var/lib/registry`.

## Env vars (auth)

| Variable | Value in manifests |
|----------|-------------------|
| `HTTP_ADDR` | `:8080` |
| `DATABASE_PATH` | `/data/registry-auth.db` |
| `REGISTRY_SERVICE` | `registry` |
| `TOKEN_ISSUER` | `registry-auth` |
| `TOKEN_CERT_PATH` | `/data/token.crt` |
| `TOKEN_KEY_PATH` | `/data/token.key` |
| `ADMIN_USER` | from Secret |
| `ADMIN_PASSWORD` | from Secret |
| `SESSION_SECRET` | from Secret |

`REGISTRY_SERVICE` and `TOKEN_ISSUER` must match `service` and `issuer` in the registry ConfigMap.

## Smoke test

```bash
curl -sf https://auth.example.com/healthz
curl -si https://registry.example.com/v2/
docker login registry.example.com
```

Admin UI: `https://auth.example.com/admin`
