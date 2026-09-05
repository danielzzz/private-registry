package auth_test

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

const testService = "registry.example.test"

func newTestIssuer(t *testing.T) (*auth.Issuer, *rsa.PrivateKey) {
	t.Helper()
	dir := t.TempDir()
	key, cert, err := auth.LoadOrCreateKeys(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := auth.NewIssuer(auth.TokenConfig{
		Issuer:  "https://" + testService,
		Service: testService,
	}, key, cert)
	if err != nil {
		t.Fatal(err)
	}
	return issuer, key
}

func newTokenHandler(t *testing.T, s *store.Store) (http.Handler, *rsa.PrivateKey) {
	t.Helper()
	issuer, key := newTestIssuer(t)
	authn := &auth.Authenticator{Store: s}
	rulesFn := func(ctx context.Context) ([]acl.Rule, error) {
		stored, err := s.ListACLRules(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]acl.Rule, len(stored))
		for i, r := range stored {
			out[i] = acl.Rule{
				SubjectKind: r.SubjectKind,
				SubjectID:   r.SubjectID,
				Pattern:     r.Pattern,
				Action:      r.Action,
			}
		}
		return out, nil
	}
	return auth.TokenHandler(authn, issuer, rulesFn, s.ListGroupIDsForUser), key
}

func tokenRequest(t *testing.T, h http.Handler, path string, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func parseTokenResponse(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal: %v body=%q", err, rec.Body.String())
	}
	token := body["token"]
	accessToken := body["access_token"]
	if token == "" {
		t.Fatalf("missing token in %v", body)
	}
	if accessToken == "" {
		t.Fatalf("missing access_token in %v", body)
	}
	if token != accessToken {
		t.Fatalf("token=%q access_token=%q want same value", token, accessToken)
	}
	return token
}

func parseJWTAccess(t *testing.T, key *rsa.PrivateKey, tokenString string) []acl.Scope {
	t.Helper()
	pubKey := key.Public().(*rsa.PublicKey)
	parsed, err := jwt.Parse(tokenString, func(_ *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	rawAccess, ok := claims["access"].([]interface{})
	if !ok {
		t.Fatalf("access=%v", claims["access"])
	}
	scopeJSON, err := json.Marshal(rawAccess)
	if err != nil {
		t.Fatal(err)
	}
	var access []acl.Scope
	if err := json.Unmarshal(scopeJSON, &access); err != nil {
		t.Fatal(err)
	}
	return access
}

func parseJWTClaims(t *testing.T, key *rsa.PrivateKey, tokenString string) jwt.MapClaims {
	t.Helper()
	pubKey := key.Public().(*rsa.PublicKey)
	parsed, err := jwt.Parse(tokenString, func(_ *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Claims.(jwt.MapClaims)
}

func TestTokenHandlerAnonymousPullAllowed(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_, err := s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectAnonymous,
		Pattern:     "public/*",
		Action:      acl.ActionPull,
	})
	if err != nil {
		t.Fatal(err)
	}

	h, key := newTokenHandler(t, s)
	path := "/token?service=" + testService + "&scope=repository:public/img:pull"
	rec := tokenRequest(t, h, path, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}

	token := parseTokenResponse(t, rec)
	access := parseJWTAccess(t, key, token)
	if len(access) != 1 || access[0].Name != "public/img" || len(access[0].Actions) != 1 || access[0].Actions[0] != "pull" {
		t.Fatalf("access=%+v", access)
	}

	claims := parseJWTClaims(t, key, token)
	if claims["sub"] != "" {
		t.Fatalf("sub=%v want empty for anonymous", claims["sub"])
	}
}

func TestTokenHandlerAnonymousPullDenied(t *testing.T) {
	s := openTestStore(t)
	h, _ := newTokenHandler(t, s)
	path := "/token?service=" + testService + "&scope=repository:private/img:pull"
	rec := tokenRequest(t, h, path, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestTokenHandlerBasicAuthPushAllowed(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	user, err := s.CreateUser(ctx, "alice", hash, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectUser,
		SubjectID:   user.ID,
		Pattern:     "danielzelisko/*",
		Action:      acl.ActionPush,
	})
	if err != nil {
		t.Fatal(err)
	}

	h, key := newTokenHandler(t, s)
	path := "/token?service=" + testService + "&scope=repository:danielzelisko/app:push"
	rec := tokenRequest(t, h, path, "alice", "s3cr3t")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}

	token := parseTokenResponse(t, rec)
	access := parseJWTAccess(t, key, token)
	if len(access) != 1 {
		t.Fatalf("access=%+v", access)
	}
	if access[0].Name != "danielzelisko/app" {
		t.Fatalf("name=%q", access[0].Name)
	}
	if len(access[0].Actions) != 2 || access[0].Actions[0] != "pull" || access[0].Actions[1] != "push" {
		t.Fatalf("actions=%v want [pull push]", access[0].Actions)
	}

	claims := parseJWTClaims(t, key, token)
	if claims["sub"] != user.ID {
		t.Fatalf("sub=%v want %s", claims["sub"], user.ID)
	}
}

func TestTokenHandlerBadPassword(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser(ctx, "alice", hash, false)
	if err != nil {
		t.Fatal(err)
	}

	h, _ := newTokenHandler(t, s)
	path := "/token?service=" + testService + "&scope=repository:danielzelisko/app:pull"
	rec := tokenRequest(t, h, path, "alice", "wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestTokenHandlerNoScopesAllowed(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	user, err := s.CreateUser(ctx, "alice", hash, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectUser,
		SubjectID:   user.ID,
		Pattern:     "other/*",
		Action:      acl.ActionPull,
	})
	if err != nil {
		t.Fatal(err)
	}

	h, _ := newTokenHandler(t, s)
	path := "/token?service=" + testService + "&scope=repository:danielzelisko/app:pull"
	rec := tokenRequest(t, h, path, "alice", "s3cr3t")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestTokenHandlerWrongService(t *testing.T) {
	s := openTestStore(t)
	h, _ := newTokenHandler(t, s)
	path := "/token?service=wrong.example.test&scope=repository:public/img:pull"
	rec := tokenRequest(t, h, path, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
