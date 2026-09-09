package store

import (
	"context"
	"time"
)

type Store interface {
	Close() error

	CreateUser(ctx context.Context, username, passwordHash string, admin bool) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	CountUsers(ctx context.Context) (int, error)
	CountActiveAdmins(ctx context.Context) (int, error)
	SetUserActive(ctx context.Context, id string, active bool) error
	SetUserAdmin(ctx context.Context, id string, admin bool) error
	UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error
	ListUsers(ctx context.Context) ([]User, error)

	CreateAPIToken(ctx context.Context, userID, name, hash string, expiresAt *time.Time) (string, error)
	ListAPITokensByUser(ctx context.Context, userID string) ([]APIToken, error)
	CountAPITokens(ctx context.Context) (int, error)
	RevokeAPIToken(ctx context.Context, id string) error

	CreateGroup(ctx context.Context, name string) (Group, error)
	AddMember(ctx context.Context, groupID, userID string) error
	RemoveMember(ctx context.Context, groupID, userID string) error
	ListGroups(ctx context.Context) ([]Group, error)
	CountGroups(ctx context.Context) (int, error)
	ListGroupIDsForUser(ctx context.Context, userID string) ([]string, error)

	CreateACLRule(ctx context.Context, input ACLRuleInput) (ACLRule, error)
	ListACLRules(ctx context.Context) ([]ACLRule, error)
	CountACLRules(ctx context.Context) (int, error)
	UpdateACLRule(ctx context.Context, rule ACLRule) error
	DeleteACLRule(ctx context.Context, id string) error
}
