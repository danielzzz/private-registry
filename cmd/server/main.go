package main

import (
	"context"
	"log"
	"net/http"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/bootstrap"
	"github.com/danielzelisko/private-registry/internal/config"
	"github.com/danielzelisko/private-registry/internal/store"
	"github.com/danielzelisko/private-registry/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	s, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	if err := bootstrap.EnsureAdmin(ctx, s, cfg.AdminUser, cfg.AdminPassword); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	key, cert, err := auth.LoadOrCreateKeys(cfg.TokenCertPath, cfg.TokenKeyPath)
	if err != nil {
		log.Fatalf("keys: %v", err)
	}

	issuer, err := auth.NewIssuer(auth.TokenConfig{
		Issuer:   cfg.TokenIssuer,
		Service:  cfg.RegistryService,
		TTL:      cfg.TokenTTL,
		CertPath: cfg.TokenCertPath,
		KeyPath:  cfg.TokenKeyPath,
	}, key, cert)
	if err != nil {
		log.Fatalf("issuer: %v", err)
	}

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
	tokenHandler := auth.TokenHandler(authn, issuer, rulesFn, s.ListGroupIDsForUser)

	handler := web.NewHandler(web.Deps{Token: tokenHandler})
	log.Printf("listening on %s", cfg.HTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, handler))
}
