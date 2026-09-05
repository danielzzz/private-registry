package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	defaultTokenTTL   = 300 * time.Second
	defaultNBFLeeway  = 60 * time.Second
)

type TokenConfig struct {
	Issuer   string
	Service  string
	TTL      time.Duration
	CertPath string
	KeyPath  string
}

type Issuer struct {
	cfg  TokenConfig
	key  *rsa.PrivateKey
	cert *x509.Certificate
	kid  string
	x5c  string
}

func NewIssuer(cfg TokenConfig, key *rsa.PrivateKey, cert *x509.Certificate) (*Issuer, error) {
	if cfg.TTL == 0 {
		cfg.TTL = defaultTokenTTL
	}
	if key == nil {
		return nil, fmt.Errorf("signing key is required")
	}
	if cert == nil {
		return nil, fmt.Errorf("certificate is required")
	}
	kid, err := LibtrustKeyID(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	return &Issuer{
		cfg:  cfg,
		key:  key,
		cert: cert,
		kid:  kid,
		x5c:  base64.StdEncoding.EncodeToString(cert.Raw),
	}, nil
}

func (i *Issuer) Mint(subject string, access []acl.Scope) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":    i.cfg.Issuer,
		"sub":    subject,
		"aud":    i.cfg.Service,
		"iat":    now.Unix(),
		"nbf":    now.Add(-defaultNBFLeeway).Unix(),
		"exp":    now.Add(i.cfg.TTL).Unix(),
		"jti":    uuid.NewString(),
		"access": access,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = i.kid
	token.Header["x5c"] = []string{i.x5c}

	return token.SignedString(i.key)
}
