#!/usr/bin/env bash
# One-command local demo: compose up, seed users/ACL, push as writer, pull as reader.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DEMO_DIR="$ROOT/examples/local-demo"
COMPOSE_ENV="$DEMO_DIR/.env"
COMPOSE=(docker compose -p private-registry-demo -f "$DEMO_DIR/docker-compose.yml")
AUTH_URL="http://127.0.0.1:18080"
REGISTRY="localhost:5000"
IMAGE="$REGISTRY/private-registry/auth:demo"
SEED_IMAGE="private-registry-demoseed:local"
VOLUME="private-registry-demo_auth-data"

WRITER_USER=writer
WRITER_PASS=writerpass
READER_USER=reader
READER_PASS=readerpass

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

wait_health() {
  local deadline=$((SECONDS + 90))
  while (( SECONDS < deadline )); do
    if curl -sf "$AUTH_URL/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  die "auth did not become healthy at $AUTH_URL/healthz within 90s"
}

ensure_env() {
  if [[ ! -f "$COMPOSE_ENV" ]]; then
    log "creating $COMPOSE_ENV from .env.example"
    cp "$DEMO_DIR/.env.example" "$COMPOSE_ENV"
  fi
}

docker_login() {
  local user=$1 pass=$2
  printf '%s\n' "$pass" | docker login "$REGISTRY" -u "$user" --password-stdin
}

main() {
  need_cmd docker
  need_cmd curl

  ensure_env
  cd "$DEMO_DIR"

  log "starting demo compose stack (auth :18080, registry :5000)"
  "${COMPOSE[@]}" up -d --build --force-recreate

  log "waiting for auth health"
  wait_health

  log "building demoseed image"
  docker build -f "$DEMO_DIR/Dockerfile.seed" -t "$SEED_IMAGE" "$ROOT"

  log "seeding demo users and ACL (auth stopped briefly for SQLite)"
  "${COMPOSE[@]}" stop auth
  docker run --rm -v "${VOLUME}:/data" "$SEED_IMAGE"
  "${COMPOSE[@]}" start auth
  wait_health

  log "building and pushing $IMAGE as $WRITER_USER"
  docker build -t "$IMAGE" "$ROOT"
  docker_login "$WRITER_USER" "$WRITER_PASS"
  docker push "$IMAGE"
  docker logout "$REGISTRY" >/dev/null

  log "pulling $IMAGE as $READER_USER"
  docker_login "$READER_USER" "$READER_PASS"
  docker pull "$IMAGE"

  log "expecting push as $READER_USER to fail"
  deny_tag="$REGISTRY/private-registry/auth:demo-should-deny"
  docker tag "$IMAGE" "$deny_tag"
  if docker push "$deny_tag" >/tmp/private-registry-demo-push.out 2>/tmp/private-registry-demo-push.err; then
    docker logout "$REGISTRY" >/dev/null || true
    die "reader was able to push; ACL may be wrong"
  fi
  docker rmi "$deny_tag" >/dev/null 2>&1 || true
  docker logout "$REGISTRY" >/dev/null || true
  log "reader push denied (expected)"

  cat <<EOF

Demo OK.

  Admin UI (admins only):
    $AUTH_URL/admin
    user/pass from examples/local-demo/.env (see .env.example)

  Registry docker login (not the Admin UI):
    docker login $REGISTRY -u $WRITER_USER -p $WRITER_PASS
    docker login $REGISTRY -u $READER_USER -p $READER_PASS

  Image: $IMAGE
  Writer: push+pull private-registry/*
  Reader: pull private-registry/*

Stop stack:
  docker compose -p private-registry-demo -f $DEMO_DIR/docker-compose.yml down

EOF
}

main "$@"
