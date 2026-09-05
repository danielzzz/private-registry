# Registry auth service implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go auth service that issues Docker Registry v2 Bearer tokens from SQLite-backed users/groups/ACLs, plus an HTMX admin UI, Compose, and single-replica k3s manifests.

**Architecture:** One binary serves `GET /token` (RS256 JWT for registry2) and cookie-session admin UI. Stock registry2 validates tokens with a shared cert. No reverse proxy of registry traffic.

**Tech Stack:** Go 1.22+, `database/sql` + `modernc.org/sqlite`, `golang.org/x/crypto/argon2`, `github.com/golang-jwt/jwt/v5`, `html/template` + HTMX + Tailwind CDN, Docker Compose, raw Kubernetes manifests.

## Global Constraints

- MIT license; module path `github.com/danielzelisko/private-registry`
- TDD on every behavior: failing test first, then minimal implementation
- SQLite only; document single-replica only
- Token auth only (not a registry reverse proxy)
- Password KDF: argon2id
- Registry JWT: RS256, claims per distribution auth JWT spec (`iss`, `sub`, `aud`, `exp`, `nbf`, `iat`, `jti`, `access`)
- Default token lifetime: 300 seconds; `nbf = iat - 60` for clock skew
- JWT header must include `kid` (libtrust-compatible fingerprint) and `x5c` (DER cert, base64)
- Empty allowed scope set → HTTP 401 (never empty-access token)
- `push` implies `pull` in ACL evaluation
- Repository patterns: `*` globs only (not regex); match repository path, not tags
- Anonymous subject allowed for pull rules only
- Admin UI: Tailwind CDN in v1; CSRF on mutating POSTs
- Bootstrap: empty DB requires `ADMIN_USER` + `ADMIN_PASSWORD` or process exits
- k3s package: raw manifests (no Helm in v1)
- No em dashes in user-facing copy or commit messages

## File structure

```text
cmd/server/main.go                 process entry, wiring
internal/config/config.go          env loading
internal/acl/match.go              glob match
internal/acl/eval.go               ACL evaluation
internal/store/store.go            DB open + migrate
internal/store/user.go
internal/store/token.go
internal/store/group.go
internal/store/acl.go
internal/auth/password.go          argon2id hash/verify
internal/auth/credentials.go       password or API token login
internal/auth/jwt.go               RS256 token issue + key load/generate
internal/auth/tokenhttp.go         GET /token handler
internal/web/server.go             mux, sessions, CSRF, rate limit
internal/web/login.go
internal/web/dashboard.go
internal/web/users.go
internal/web/apitokens.go
internal/web/groups.go
internal/web/aclrules.go
internal/web/templates/*.html
deploy/compose/docker-compose.yml
deploy/compose/registry-config.yml
deploy/k8s/*.yaml
LICENSE
README.md
go.mod
```

Tests live next to packages as `*_test.go`.

---

### Task 1: Module skeleton and healthz

**Files:**
- Create: `go.mod`, `cmd/server/main.go`, `internal/web/server.go`, `internal/web/server_test.go`
- Create: `LICENSE` (MIT)

**Interfaces:**
- Produces: `web.New(mux wiring later) *http.ServeMux` with at least `GET /healthz` → `200` body `ok`
- Consumes: nothing

- [ ] **Step 1: Write the failing test**

```go
// internal/web/server_test.go
package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielzelisko/private-registry/internal/web"
)

func TestHealthz(t *testing.T) {
	h := web.NewHandler(web.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /media/daniel/data1/lindata/proyectos/private-registry && go test ./internal/web/ -run TestHealthz -v`
Expected: FAIL (package or `NewHandler` undefined)

- [ ] **Step 3: Minimal implementation**

```bash
go mod init github.com/danielzelisko/private-registry
```

```go
// internal/web/server.go
package web

import "net/http"

type Deps struct{}

func NewHandler(_ Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}
```

```go
// cmd/server/main.go
package main

import (
	"log"
	"net/http"

	"github.com/danielzelisko/private-registry/internal/web"
)

func main() {
	addr := ":8080"
	log.Fatal(http.ListenAndServe(addr, web.NewHandler(web.Deps{})))
}
```

Add MIT `LICENSE` with copyright holder suitable for the author.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/web/ -run TestHealthz -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod LICENSE cmd/server/main.go internal/web/server.go internal/web/server_test.go
git commit -m "feat: add module skeleton and healthz endpoint"
```

---

### Task 2: Repository glob matching

**Files:**
- Create: `internal/acl/match.go`, `internal/acl/match_test.go`

**Interfaces:**
- Produces: `func Match(pattern, name string) bool`
- Consumes: nothing

- [ ] **Step 1: Write the failing tests**

```go
// internal/acl/match_test.go
package acl_test

import (
	"testing"

	"github.com/danielzelisko/private-registry/internal/acl"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"danielzelisko/test-project*", "danielzelisko/test-project", true},
		{"danielzelisko/test-project*", "danielzelisko/test-project-app", true},
		{"danielzelisko/test-project*", "danielzelisko/other", false},
		{"library/*", "library/nginx", true},
		{"library/*", "library/nginx/extra", false},
		{"*", "thing", true},
		{"*", "any/thing", false},
		{"*/*", "any/thing", true},
		{"exact/name", "exact/name", true},
		{"exact/name", "exact/other", false},
	}
	for _, tc := range cases {
		got := acl.Match(tc.pattern, tc.name)
		if got != tc.want {
			t.Fatalf("Match(%q,%q)=%v want %v", tc.pattern, tc.name, got, tc.want)
		}
	}
}
```

v1 uses Go `path.Match` semantics: `*` does not cross `/`. So `danielzelisko/test-project*` matches `danielzelisko/test-project-app`, and `library/*` matches one path segment.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/acl/ -run TestMatch -v`
Expected: FAIL (`Match` undefined)

- [ ] **Step 3: Minimal implementation**

```go
// internal/acl/match.go
package acl

import "path"

func Match(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/acl/ -run TestMatch -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/acl/match.go internal/acl/match_test.go
git commit -m "feat: add repository path glob matching"
```

---

### Task 3: ACL evaluation

**Files:**
- Create: `internal/acl/eval.go`, `internal/acl/eval_test.go`, `internal/acl/types.go`

**Interfaces:**
- Produces:
  - `type Action string` with `ActionPull`, `ActionPush`
  - `type SubjectKind string` with `SubjectUser`, `SubjectGroup`, `SubjectAnonymous`
  - `type Rule struct { SubjectKind; SubjectID string; Pattern string; Action Action }`
  - `type Scope struct { Type string; Name string; Actions []string }`
  - `type Identity struct { UserID string; GroupIDs []string; Anonymous bool }`
  - `func Evaluate(id Identity, rules []Rule, requested []Scope) []Scope`  
    Returns only allowed scopes. For a requested repo with `push`, if only pull is granted, return pull only. If push granted, include both pull and push in actions. Drop scopes with zero actions.
- Consumes: `acl.Match`

- [ ] **Step 1: Write the failing tests**

```go
func TestEvaluate_UserDirectPull(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectUser, SubjectID: "u1",
		Pattern: "danielzelisko/test-project*", Action: acl.ActionPull,
	}}
	id := acl.Identity{UserID: "u1"}
	req := []acl.Scope{{Type: "repository", Name: "danielzelisko/test-project-app", Actions: []string{"pull"}}}
	got := acl.Evaluate(id, rules, req)
	if len(got) != 1 || got[0].Name != req[0].Name || !contains(got[0].Actions, "pull") {
		t.Fatalf("got=%v", got)
	}
}

func TestEvaluate_PushImpliesPull(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectUser, SubjectID: "u1",
		Pattern: "app/*", Action: acl.ActionPush,
	}}
	id := acl.Identity{UserID: "u1"}
	req := []acl.Scope{{Type: "repository", Name: "app/web", Actions: []string{"push", "pull"}}}
	got := acl.Evaluate(id, rules, req)
	if len(got) != 1 || !contains(got[0].Actions, "push") || !contains(got[0].Actions, "pull") {
		t.Fatalf("got=%v", got)
	}
}

func TestEvaluate_GroupAndAnonymous(t *testing.T) {
	rules := []acl.Rule{
		{SubjectKind: acl.SubjectGroup, SubjectID: "g1", Pattern: "team/*", Action: acl.ActionPull},
		{SubjectKind: acl.SubjectAnonymous, Pattern: "public/*", Action: acl.ActionPull},
	}
	anon := acl.Evaluate(acl.Identity{Anonymous: true}, rules, []acl.Scope{
		{Type: "repository", Name: "public/foo", Actions: []string{"pull"}},
		{Type: "repository", Name: "team/bar", Actions: []string{"pull"}},
	})
	if len(anon) != 1 || anon[0].Name != "public/foo" {
		t.Fatalf("anon=%v", anon)
	}
	member := acl.Evaluate(acl.Identity{UserID: "u1", GroupIDs: []string{"g1"}}, rules, []acl.Scope{
		{Type: "repository", Name: "team/bar", Actions: []string{"pull"}},
	})
	if len(member) != 1 {
		t.Fatalf("member=%v", member)
	}
}

func TestEvaluate_DenyByOmission(t *testing.T) {
	got := acl.Evaluate(acl.Identity{UserID: "u1"}, nil, []acl.Scope{
		{Type: "repository", Name: "x/y", Actions: []string{"pull"}},
	})
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}
```

Helper `contains` in the test file.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/acl/ -run TestEvaluate -v`
Expected: FAIL

- [ ] **Step 3: Implement `types.go` + `eval.go`**

Evaluation algorithm:

1. For each requested scope with `Type == "repository"`, start with empty allowed action set.
2. For each rule whose subject matches identity (anonymous flag, user id, or group id), if `Match(rule.Pattern, scope.Name)`:
   - if rule.Action == pull → allow pull
   - if rule.Action == push → allow pull and push
3. Intersect allowed actions with **requested** actions on that scope (only grant what was asked, except when push was asked and rule grants push, include pull if requested OR always include pull when push granted and pull was requested — follow requested list intersection).
4. If push is allowed and pull was requested (or push implies pull always in token): include both when the rule grants push and the request asked for either. Spec: push implies pull. When building token actions for a scope, if push allowed, emit `["pull","push"]` filtered to those that appear in the request **or** always add pull when push is present in the result. Practical registry behavior: if client asks `push,pull` and user has push, return both. If client asks only `push` and user has push, return `push` and also `pull` (implied). Implement: if push granted → result actions include push and pull; then if we want intersection with request, still force-add pull when push present.

Clear rule for implementers:

```text
grantedPull, grantedPush := ...
out := []string{}
if grantedPush {
  out = append(out, "pull", "push")
} else if grantedPull {
  out = append(out, "pull")
}
// keep only actions that were requested, BUT if "push" remains, ensure "pull" is present
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/acl/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/acl/
git commit -m "feat: evaluate ACL rules for registry scopes"
```

---

### Task 4: SQLite store — users and bootstrap shape

**Files:**
- Create: `internal/store/store.go`, `internal/store/migrate.go`, `internal/store/user.go`, `internal/store/user_test.go`

**Interfaces:**
- Produces:
  - `func Open(path string) (*Store, error)`
  - `func (s *Store) Close() error`
  - `type User struct { ID, Username, PasswordHash string; Active, Admin bool }`
  - `func (s *Store) CreateUser(ctx, username, passwordHash string, admin bool) (User, error)`
  - `func (s *Store) GetUserByUsername(ctx, username string) (User, error)`
  - `func (s *Store) CountUsers(ctx) (int, error)`
  - `func (s *Store) SetUserActive`, `SetUserAdmin`, `UpdatePasswordHash`, `ListUsers`
- Consumes: `modernc.org/sqlite` driver

- [ ] **Step 1: Write failing integration test** using temp DB file

```go
func TestCreateAndGetUser(t *testing.T) {
	s := openTestDB(t)
	u, err := s.CreateUser(context.Background(), "alice", "hash", true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByUsername(context.Background(), "alice")
	if err != nil || got.ID != u.ID || !got.Admin || !got.Active {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/store/ -run TestCreateAndGetUser -v`

- [ ] **Step 3: Implement migrations + user methods**

Schema (in `migrate.go`):

```sql
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  admin INTEGER NOT NULL DEFAULT 0
);
```

Use UUID strings for IDs (`github.com/google/uuid` or `crypto/rand` hex).

- [ ] **Step 4: Tests PASS**

Run: `go test ./internal/store/ -v`

- [ ] **Step 5: Commit**

```bash
git add internal/store/ go.mod go.sum
git commit -m "feat: add SQLite store and users table"
```

---

### Task 5: Store — API tokens, groups, ACL rules

**Files:**
- Create: `internal/store/token.go`, `internal/store/group.go`, `internal/store/acl.go`, and `*_test.go` for each

**Interfaces:**
- Produces:
  - API tokens: `CreateAPIToken(userID, name, hash string, expiresAt *time.Time) (id string, err)`, `ListAPITokensByUser`, `RevokeAPIToken`, `FindActiveTokenByPrefix` or verify by scanning user tokens (prefer store method `GetUserAPITokens` + hash check in auth layer; optional `LookupAPIToken(username, secret)` that returns user if hash matches)
  - Groups: `CreateGroup`, `AddMember`, `RemoveMember`, `ListGroups`, `ListGroupIDsForUser`
  - ACL: `type ACLRule` with fields matching `acl.Rule` plus `ID`; `CreateACLRule`, `ListACLRules`, `DeleteACLRule`, `UpdateACLRule`
  - Subject: store `subject_kind` as `user|group|anonymous`; `subject_id` nullable for anonymous
- Consumes: `Store`

Schema additions:

```sql
CREATE TABLE api_tokens (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  token_hash TEXT NOT NULL,
  expires_at INTEGER,
  active INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE groups (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE group_members (
  group_id TEXT NOT NULL REFERENCES groups(id),
  user_id TEXT NOT NULL REFERENCES users(id),
  PRIMARY KEY (group_id, user_id)
);
CREATE TABLE acl_rules (
  id TEXT PRIMARY KEY,
  subject_kind TEXT NOT NULL,
  subject_id TEXT,
  pattern TEXT NOT NULL,
  action TEXT NOT NULL
);
```

- [ ] **Step 1: Failing tests** for create group + member → `ListGroupIDsForUser`; create ACL rule → list; create API token → list/revoke

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement migrations and methods**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: store API tokens, groups, and ACL rules"
```

---

### Task 6: Password hashing and credential verification

**Files:**
- Create: `internal/auth/password.go`, `internal/auth/password_test.go`, `internal/auth/credentials.go`, `internal/auth/credentials_test.go`

**Interfaces:**
- Produces:
  - `func HashPassword(password string) (string, error)`
  - `func CheckPassword(hash, password string) bool`
  - `type Authenticator struct { Store *store.Store }`
  - `func (a *Authenticator) Authenticate(ctx, username, secret string) (store.User, error)`  
    Try password hash first; if fail, try active API tokens for that user (constant-time where practical). Reject inactive users.
  - API token plaintext format: `prt_<id>_<secret>` or random 32-byte hex stored hashed; on create, return plaintext once. Verification: look up by username, compare hash of provided secret against each active token (or encode token id in plaintext for O(1) lookup: `prt_<tokenID>.<secret>`).
- Consumes: `store.Store` token + user methods

Locked token plaintext format: `prt_<tokenID>_<32-byte-hex-secret>`. Store only hash of the full plaintext or of the secret part; verify by parsing tokenID and loading that row.

- [ ] **Step 1: Failing tests** for hash round-trip; authenticate with password; authenticate with API token; reject bad secret; reject inactive user

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement argon2id** (params: time=1, memory=64MB, threads=4, keyLen=32; encode salt+hash in PHC-style string or use a small helper). Implement `Authenticate`.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/auth/ go.mod go.sum
git commit -m "feat: add argon2id passwords and credential auth"
```

---

### Task 7: JWT issuer (registry tokens)

**Files:**
- Create: `internal/auth/jwt.go`, `internal/auth/jwt_test.go`, `internal/auth/keys.go`

**Interfaces:**
- Produces:
  - `type TokenConfig struct { Issuer string; Service string; TTL time.Duration; CertPath, KeyPath string }`
  - `type Issuer struct { ... }`
  - `func LoadOrCreateKeys(certPath, keyPath string) (*rsa.PrivateKey, *x509.Certificate, error)`
  - `func NewIssuer(cfg TokenConfig, key *rsa.PrivateKey, cert *x509.Certificate) *Issuer`
  - `func (i *Issuer) Mint(subject string, access []acl.Scope) (tokenString string, err error)`
  - Claims must include `access` as in distribution spec; `aud` = service name; anonymous subject `""`
  - Header: `alg=RS256`, `kid` = libtrust key id for the public key, `x5c` = [base64 DER of cert]
- Consumes: `acl.Scope`

Kid algorithm: implement the Docker libtrust key ID (MD5 of DER public key with `:` separators) used by distribution — copy well-known algorithm from docker/libtrust or distribution docs so registry accepts the token.

- [ ] **Step 1: Failing test** — mint token, parse with `jwt.Parse` using public key, assert claims `iss`, `aud`, `access`, `exp-iat ≈ 300`, header `x5c` present

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement key generate (RSA 4096 or 2048 for tests/dev), PEM write, mint**

Dependency: `github.com/golang-jwt/jwt/v5`

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/auth/
git commit -m "feat: issue RS256 registry JWTs with kid and x5c"
```

---

### Task 8: `GET /token` HTTP handler

**Files:**
- Create: `internal/auth/tokenhttp.go`, `internal/auth/tokenhttp_test.go`
- Modify: `internal/web/server.go` to mount token handler from Deps

**Interfaces:**
- Produces: `func TokenHandler(authn *Authenticator, issuer *Issuer, rules func(ctx) ([]acl.Rule, error), groups func(ctx, userID string) ([]string, error)) http.Handler`
- Query: `service`, `scope` (repeatable or space-separated; support multiple `scope` query params). Parse `repository:name:action` or `repository:name:pull,push`.
- Auth: optional Basic. If missing → identity anonymous. If present → `Authenticate`.
- If `service` != configured service → 401
- Evaluate ACL; if no access → 401
- Response JSON: `{"token":"..."}` (also accept `access_token` field duplicate for compatibility: include both `token` and `access_token` same value)
- Consumes: Authenticator, Issuer, store list rules / groups

- [ ] **Step 1: Failing httptest tests**

  - anonymous pull allowed by rule → 200 + JWT with pull
  - anonymous pull denied → 401
  - basic auth push allowed → 200 with push+pull
  - bad password → 401
  - no scopes allowed → 401

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement handler + wire in `web.Deps`**

```go
type Deps struct {
	Token http.Handler // optional; if nil, skip
}
```

Mount `GET /token`.

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/auth/ internal/web/
git commit -m "feat: add registry token HTTP endpoint"
```

---

### Task 9: Config, bootstrap, main wiring

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `config.Load() (Config, error)` from env vars listed in the spec
- On start: open DB, migrate, if `CountUsers==0` then require admin env and create admin with hashed password, load/create keys, build Authenticator + Issuer + web handler
- Default `HTTP_ADDR=:8080`, `TOKEN_TTL` optional override default 300s

- [ ] **Step 1: Failing test** for missing admin on empty DB path — test bootstrap function `store.BootstrapAdmin` / `bootstrap.EnsureAdmin` returns error when env empty and count 0; succeeds when set

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement**

- [ ] **Step 4: PASS** + manual `go run ./cmd/server` with env against temp dir

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go internal/config/
git commit -m "feat: wire config, bootstrap admin, and server main"
```

---

### Task 10: Admin sessions, login/logout, CSRF

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/session.go`, `internal/web/login.go`, `internal/web/csrf.go`, templates under `internal/web/templates/`, tests

**Interfaces:**
- Cookie name `session`; value signed with `SESSION_SECRET` (use `gorilla/sessions` or a tiny HMAC token containing user id)
- Only `Admin==true` users may access `/admin/*`
- CSRF: hidden field token in forms; validate on POST
- Rate limit: simple per-IP memory counter on `POST /login` and `GET /token` (e.g. 20/min)

- [ ] **Step 1: Failing tests** — login success sets cookie; non-admin cannot open `/admin/`; POST without CSRF fails; logout clears session

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement login page (Tailwind CDN), session middleware, CSRF middleware**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/web/
git commit -m "feat: add admin login sessions and CSRF"
```

---

### Task 11: Admin UI — users and dashboard

**Files:**
- Create: `internal/web/dashboard.go`, `internal/web/users.go`, templates
- Tests with sqlite + httptest

**Interfaces:**
- `GET /admin/` dashboard counts
- Users CRUD-ish: list, create, disable/enable, reset password, toggle admin
- HTMX partials optional; full page OK for v1

- [ ] **Step 1: Failing tests** for create user via POST and see in list; disable user

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/web/ internal/store/
git commit -m "feat: add admin dashboard and user management"
```

---

### Task 12: Admin UI — API tokens, groups, ACL rules

**Files:**
- Create: `internal/web/apitokens.go`, `internal/web/groups.go`, `internal/web/aclrules.go`, templates, tests

**Interfaces:**
- Create token → show plaintext once
- Revoke token
- Groups create + add/remove members
- ACL rules create/delete (edit = delete+create OK for v1); subject picker user/group/anonymous; action pull/push; pattern text field

- [ ] **Step 1: Failing tests** covering one happy path each (token create returns secret once; group membership; ACL rule appears in list and affects a unit Evaluate using listed rules — optional HTTP-level check that rule is stored)

- [ ] **Step 2: FAIL**

- [ ] **Step 3: Implement all three admin sections**

- [ ] **Step 4: PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/web/
git commit -m "feat: admin UI for tokens, groups, and ACL rules"
```

---

### Task 13: Compose stack and registry config example

**Files:**
- Create: `deploy/compose/docker-compose.yml`, `deploy/compose/registry-config.yml`, `deploy/compose/.env.example`, `deploy/compose/README.md`

**Interfaces:**
- Services `auth` and `registry`
- Shared volume for certs: auth writes cert/key; registry mounts cert as `rootcertbundle`
- Registry env/config:
  - `realm: http://auth:8080/token` (or host-mapped URL for real docker clients on host — document host networking / published ports)
  - `service: registry`
  - `issuer: registry-auth` matching `TOKEN_ISSUER`
- Auth publishes `8080`; registry publishes `5000`
- Note: for `docker login localhost:5000` from host, realm must be reachable from host (e.g. `http://127.0.0.1:8080/token`)

- [ ] **Step 1: Write compose files and short smoke steps in README** (no automated docker test required in CI for v1)

- [ ] **Step 2: Bring stack up and smoke if Docker available**

Run: `docker compose -f deploy/compose/docker-compose.yml up -d --build`
Expected: both healthy; `/healthz` ok

- [ ] **Step 3: Commit**

```bash
git add deploy/compose/
git commit -m "feat: add Compose stack for auth and registry2"
```

---

### Task 14: k3s raw manifests

**Files:**
- Create: `deploy/k8s/namespace.yaml`, `deployment-auth.yaml`, `pvc.yaml`, `service-auth.yaml`, `deployment-registry.yaml`, `service-registry.yaml`, `ingress.yaml`, `secret.example.yaml`, `README.md`

**Interfaces:**
- Auth Deployment replicas: 1
- PVC mounted at data path for SQLite + keys
- Secrets for `ADMIN_USER`, `ADMIN_PASSWORD`, `SESSION_SECRET`
- Ingress hosts documented as placeholders `registry.example.com` and realm URL
- README warns: single replica only for SQLite

- [ ] **Step 1: Author manifests**

- [ ] **Step 2: `kubectl apply --dry-run=client` if kubectl available**

- [ ] **Step 3: Commit**

```bash
git add deploy/k8s/
git commit -m "feat: add single-replica k3s manifests"
```

---

### Task 15: Root README and polish

**Files:**
- Create: `README.md`
- Modify: any gaps found (`.gitignore` for `data/`, `*.pem`, binaries)

**README must cover:** what it is, quick Compose start, env vars table, ACL glob examples, anonymous pull, API token login, k3s single-replica note, MIT.

- [ ] **Step 1: Write README + .gitignore**

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add README.md .gitignore
git commit -m "docs: add README and gitignore"
```

---

## Spec coverage checklist

| Spec item | Task |
|-----------|------|
| Token auth / JWT | 7, 8 |
| Users + API tokens | 4, 5, 6, 12 |
| Groups + ACL wildcards | 2, 3, 5, 12 |
| Anonymous pull | 3, 8 |
| Admin UI pages | 10–12 |
| SQLite + bootstrap | 4, 9 |
| Compose + k3s | 13, 14 |
| TDD | all feature tasks |
| healthz | 1 |
| Rate limit + CSRF | 10 |
| MIT | 1, 15 |

## Plan self-review notes

- Locked open decisions: argon2id, raw k8s manifests, 300s TTL, 60s nbf skew, Tailwind CDN, `prt_<id>_<secret>` token format, `path.Match` glob semantics (`*` does not cross `/`)
- `test-project*` still matches `test-project-app` because `*` is within one path segment after the `/`
- JWT `kid` + `x5c` required for modern distribution compatibility
