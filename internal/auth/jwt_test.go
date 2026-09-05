package auth_test

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

func TestIssuerMintProducesValidRegistryJWT(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	key, cert, err := auth.LoadOrCreateKeys(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := auth.TokenConfig{
		Issuer:  "https://registry.example.test",
		Service: "registry.example.test",
		TTL:     300 * time.Second,
	}
	issuer, err := auth.NewIssuer(cfg, key, cert)
	if err != nil {
		t.Fatal(err)
	}

	access := []acl.Scope{
		{Type: "repository", Name: "danielzelisko/app", Actions: []string{"pull", "push"}},
	}
	tokenString, err := issuer.Mint("user-123", access)
	if err != nil {
		t.Fatal(err)
	}

	pubKey := key.Public().(*rsa.PublicKey)
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			t.Fatalf("alg=%s", token.Method.Alg())
		}
		return pubKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Valid {
		t.Fatal("token not valid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}

	if claims["iss"] != cfg.Issuer {
		t.Fatalf("iss=%v want %s", claims["iss"], cfg.Issuer)
	}
	if claims["sub"] != "user-123" {
		t.Fatalf("sub=%v", claims["sub"])
	}
	if claims["aud"] != cfg.Service {
		t.Fatalf("aud=%v want %s", claims["aud"], cfg.Service)
	}
	if claims["jti"] == nil || claims["jti"] == "" {
		t.Fatal("missing jti")
	}

	iat, err := claims.GetIssuedAt()
	if err != nil {
		t.Fatal(err)
	}
	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatal(err)
	}
	nbf, err := claims.GetNotBefore()
	if err != nil {
		t.Fatal(err)
	}

	ttl := exp.Time.Sub(iat.Time).Seconds()
	if math.Abs(ttl-300) > 2 {
		t.Fatalf("exp-iat=%v want ~300", ttl)
	}
	nbfSkew := iat.Time.Sub(nbf.Time).Seconds()
	if math.Abs(nbfSkew-60) > 2 {
		t.Fatalf("iat-nbf=%v want ~60", nbfSkew)
	}

	rawAccess, ok := claims["access"].([]interface{})
	if !ok || len(rawAccess) != 1 {
		t.Fatalf("access=%v", claims["access"])
	}
	scopeJSON, err := json.Marshal(rawAccess[0])
	if err != nil {
		t.Fatal(err)
	}
	var scope acl.Scope
	if err := json.Unmarshal(scopeJSON, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.Type != "repository" || scope.Name != "danielzelisko/app" {
		t.Fatalf("scope=%+v", scope)
	}
	if len(scope.Actions) != 2 || scope.Actions[0] != "pull" || scope.Actions[1] != "push" {
		t.Fatalf("actions=%v", scope.Actions)
	}

	x5c, ok := parsed.Header["x5c"].([]interface{})
	if !ok || len(x5c) != 1 {
		t.Fatalf("x5c header=%v", parsed.Header["x5c"])
	}
	x5cStr, ok := x5c[0].(string)
	if !ok {
		t.Fatalf("x5c[0] type=%T", x5c[0])
	}
	der, err := base64.StdEncoding.DecodeString(x5cStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(der) == 0 {
		t.Fatal("empty x5c DER")
	}

	kid, ok := parsed.Header["kid"].(string)
	if !ok || kid == "" {
		t.Fatalf("kid=%v", parsed.Header["kid"])
	}
	expectedKid, err := auth.LibtrustKeyID(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if kid != expectedKid {
		t.Fatalf("kid=%q want %q", kid, expectedKid)
	}
}

func TestIssuerMintAllowsAnonymousSubject(t *testing.T) {
	dir := t.TempDir()
	key, cert, err := auth.LoadOrCreateKeys(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := auth.NewIssuer(auth.TokenConfig{
		Issuer:  "https://registry.example.test",
		Service: "registry.example.test",
	}, key, cert)
	if err != nil {
		t.Fatal(err)
	}

	tokenString, err := issuer.Mint("", []acl.Scope{{Type: "repository", Name: "public/img", Actions: []string{"pull"}}})
	if err != nil {
		t.Fatal(err)
	}

	pubKey := key.Public().(*rsa.PublicKey)
	parsed, err := jwt.Parse(tokenString, func(_ *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "" {
		t.Fatalf("sub=%v want empty", claims["sub"])
	}
}

func TestIssuerMintAccessClaimUsesLowercaseJSONKeys(t *testing.T) {
	dir := t.TempDir()
	key, cert, err := auth.LoadOrCreateKeys(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := auth.NewIssuer(auth.TokenConfig{
		Issuer:  "https://registry.example.test",
		Service: "registry.example.test",
	}, key, cert)
	if err != nil {
		t.Fatal(err)
	}

	tokenString, err := issuer.Mint("user-1", []acl.Scope{
		{Type: "repository", Name: "danielzelisko/app", Actions: []string{"pull"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts=%d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	rawAccess, ok := claims["access"]
	if !ok {
		t.Fatal("missing access claim")
	}

	var access []map[string]json.RawMessage
	if err := json.Unmarshal(rawAccess, &access); err != nil {
		t.Fatal(err)
	}
	if len(access) != 1 {
		t.Fatalf("access len=%d", len(access))
	}
	scope := access[0]
	for _, key := range []string{"type", "name", "actions"} {
		if _, ok := scope[key]; !ok {
			t.Fatalf("missing lowercase key %q in access scope: %v", key, scope)
		}
	}
	for k := range scope {
		if k != "type" && k != "name" && k != "actions" {
			t.Fatalf("unexpected key %q in access scope", k)
		}
	}
}

func TestNewIssuerReturnsErrorWhenKeyIDFails(t *testing.T) {
	dir := t.TempDir()
	_, cert, err := auth.LoadOrCreateKeys(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = auth.NewIssuer(auth.TokenConfig{
		Issuer:  "https://registry.example.test",
		Service: "registry.example.test",
	}, nil, cert)
	if err == nil {
		t.Fatal("expected error when key is nil")
	}
}
