package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielzelisko/private-registry/internal/acl"
)

type RulesFunc func(ctx context.Context) ([]acl.Rule, error)
type GroupsFunc func(ctx context.Context, userID string) ([]string, error)

func TokenHandler(authn *Authenticator, issuer *Issuer, rules RulesFunc, groups GroupsFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		service := r.URL.Query().Get("service")
		if service != issuer.Service() {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		requested := parseScopeQuery(r.URL.Query()["scope"])

		var identity acl.Identity
		var subject string

		username, password, hasBasic := r.BasicAuth()
		if hasBasic {
			user, err := authn.Authenticate(ctx, username, password)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			groupIDs, err := groups(ctx, user.ID)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			identity = acl.Identity{UserID: user.ID, GroupIDs: groupIDs, Anonymous: false}
			subject = user.ID
		} else {
			identity = acl.Identity{Anonymous: true}
		}

		ruleList, err := rules(ctx)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		access := acl.Evaluate(identity, ruleList, requested)
		if len(access) == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := issuer.Mint(subject, access)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token":        token,
			"access_token": token,
		})
	})
}

func parseScopeQuery(params []string) []acl.Scope {
	var scopes []acl.Scope
	for _, param := range params {
		for _, part := range strings.Fields(param) {
			scope, ok := parseScope(part)
			if ok {
				scopes = append(scopes, scope)
			}
		}
	}
	return scopes
}

func parseScope(s string) (acl.Scope, bool) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 || parts[0] != "repository" || parts[1] == "" || parts[2] == "" {
		return acl.Scope{}, false
	}
	actions := strings.Split(parts[2], ",")
	return acl.Scope{
		Type:    "repository",
		Name:    parts[1],
		Actions: actions,
	}, true
}
