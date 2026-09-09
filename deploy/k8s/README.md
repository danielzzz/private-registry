# k3s manifests (auth + registry2)

Raw Kubernetes manifests for a single-replica private registry stack on k3s.

## Prerequisites

- k3s (or any Kubernetes cluster with an Ingress controller)
- External MySQL database reachable from the cluster, with a dedicated user that has DDL and DML on the `registry_auth` schema (or the database named in your DSN)
- Replace placeholders before applying:
  - image in `deployment-auth.yaml` (build from this repo's `Dockerfile` and push to your registry)
  - `<STORAGE_CLASS>` in `pvc.yaml` (k3s default: `local-path`)
  - `registry.example.com` and `auth.example.com` in `ingress.yaml`
  - `realm` URL in `deployment-registry.yaml` ConfigMap (must match the auth Ingress host)
  - strong values in `secret.yaml` (never reuse the example passwords)
  - `DATABASE_DSN` in `secret.yaml` pointing at your external MySQL instance

## Install

```bash
# 1. Create secrets (do not commit secret.yaml)
cp secret.example.yaml secret.yaml
# edit secret.yaml: ADMIN_USER, ADMIN_PASSWORD, SESSION_SECRET, DATABASE_DSN

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

## Signing certificates

Paths in the default manifests: `TOKEN_CERT_PATH=/data/token.crt`, `TOKEN_KEY_PATH=/data/token.key` on the auth PVC.

### Auto (default)

On first boot, if the key file is missing, auth writes `token.crt` and `token.key` under `/data` on the PVC. Later pods reuse them.

Apply and start auth first. Wait until it is healthy (`kubectl wait` or `curl .../healthz`) before relying on registry token auth.

If the registry starts before those certs exist, it may fail briefly. Restart the registry pod once auth is healthy; that usually recovers. Mount the same cert into the registry container as `rootcertbundle` (see the registry ConfigMap / volume in `deployment-registry.yaml`).

### Provide certs via a Kubernetes Secret

Yes. Create a Secret with the PEM files, mount them into auth (and the cert into registry), and point `TOKEN_*_PATH` at the mount paths. Auth will load existing files and skip generation.

```bash
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout token.key -out token.crt -days 3650 \
  -subj "/CN=private-registry"
chmod 600 token.key

kubectl -n private-registry create secret generic registry-auth-signing \
  --from-file=token.crt=./token.crt \
  --from-file=token.key=./token.key
```

Example auth pod fragments (merge into `deployment-auth.yaml`):

```yaml
env:
  - name: TOKEN_CERT_PATH
    value: /certs/token.crt
  - name: TOKEN_KEY_PATH
    value: /certs/token.key
volumeMounts:
  - name: signing
    mountPath: /certs
    readOnly: true
volumes:
  - name: signing
    secret:
      secretName: registry-auth-signing
      defaultMode: 0400
```

Mount `token.crt` from the same Secret into the registry Deployment at the path used by `rootcertbundle` in the registry config. Keep the auth PVC for other data if needed, or drop key paths from the PVC once secrets own the PEMs.

Rotating the Secret requires restarting auth and registry; clients must get new tokens after the key changes.

## Single replica only

**Do not scale the auth Deployment beyond 1 replica** unless every replica mounts the **same** signing key (for example the Secret above). Auto-generated keys on a single `ReadWriteOnce` PVC are not safe for multi-replica auth.

Registry image blobs use `emptyDir` in these manifests (ephemeral). For production, add a separate PVC for `/var/lib/registry`.

## Env vars (auth)

| Variable | Value in manifests |
|----------|-------------------|
| `HTTP_ADDR` | `:8080` |
| `DATABASE_DRIVER` | `mysql` |
| `DATABASE_DSN` | from Secret |
| `REGISTRY_SERVICE` | `registry` |
| `TOKEN_ISSUER` | `registry-auth` |
| `TOKEN_CERT_PATH` | `/data/token.crt` |
| `TOKEN_KEY_PATH` | `/data/token.key` |
| `ADMIN_USER` | from Secret |
| `ADMIN_PASSWORD` | from Secret |
| `SESSION_SECRET` | from Secret |

`REGISTRY_SERVICE` and `TOKEN_ISSUER` must match `service` and `issuer` in the registry ConfigMap.

## Cutover from SQLite

If you are moving an existing SQLite deployment to MySQL, run the migration tool before flipping the driver:

```bash
go run ./cmd/migrate-sqlite-to-mysql \
  -sqlite /path/to/registry-auth.db \
  -mysql 'authuser:password@tcp(mysql.example.com:3306)/registry_auth?parseTime=true'
```

The target MySQL database must be empty. After a successful migration, update `secret.yaml` with the DSN and apply the updated auth Deployment (`DATABASE_DRIVER=mysql`).

## Smoke test

```bash
curl -sf https://auth.example.com/healthz
curl -si https://registry.example.com/v2/
docker login registry.example.com
```

Admin UI: `https://auth.example.com/admin`
