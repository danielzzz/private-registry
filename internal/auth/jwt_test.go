package auth_test

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math"
	"path/filepath"
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
	issuer := auth.NewIssuer(cfg, key, cert)

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
}

func TestIssuerMintAllowsAnonymousSubject(t *testing.T) {
	dir := t.TempDir()
	key, cert, err := auth.LoadOrCreateKeys(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	issuer := auth.NewIssuer(auth.TokenConfig{
		Issuer:  "https://registry.example.test",
		Service: "registry.example.test",
	}, key, cert)

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
