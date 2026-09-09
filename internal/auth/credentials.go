package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/danielzzz/private-registry/internal/store"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user inactive")
)

type Authenticator struct {
	Store store.Store
}

func (a *Authenticator) Authenticate(ctx context.Context, username, secret string) (store.User, error) {
	user, err := a.Store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.User{}, ErrInvalidCredentials
		}
		return store.User{}, err
	}
	if !user.Active {
		return store.User{}, ErrUserInactive
	}
	if CheckPassword(user.PasswordHash, secret) {
		return user, nil
	}
	if ok, err := a.authenticateAPIToken(ctx, user, secret); err != nil {
		return store.User{}, err
	} else if ok {
		return user, nil
	}
	return store.User{}, ErrInvalidCredentials
}

func (a *Authenticator) authenticateAPIToken(ctx context.Context, user store.User, secret string) (bool, error) {
	tokenID, tokenSecret, ok := parseAPIToken(secret)
	if !ok {
		return false, nil
	}

	tokens, err := a.Store.ListAPITokensByUser(ctx, user.ID)
	if err != nil {
		return false, err
	}

	now := time.Now()
	for _, token := range tokens {
		if token.ID != tokenID || !token.Active {
			continue
		}
		if token.ExpiresAt != nil && now.After(*token.ExpiresAt) {
			continue
		}
		if CheckPassword(token.TokenHash, tokenSecret) {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

func parseAPIToken(s string) (tokenID, tokenSecret string, ok bool) {
	if !strings.HasPrefix(s, "prt_") {
		return "", "", false
	}
	rest := strings.TrimPrefix(s, "prt_")
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	if len(parts[1]) != 64 {
		return "", "", false
	}
	for _, c := range parts[1] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return "", "", false
		}
	}
	return parts[0], parts[1], true
}
