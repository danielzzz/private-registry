package store

import (
	"time"

	"github.com/danielzelisko/private-registry/internal/acl"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Active       bool
	Admin        bool
}

type APIToken struct {
	ID        string
	UserID    string
	Name      string
	TokenHash string
	ExpiresAt *time.Time
	Active    bool
}

type Group struct {
	ID   string
	Name string
}

type ACLRule struct {
	ID          string
	SubjectKind acl.SubjectKind
	SubjectID   string
	Pattern     string
	Action      acl.Action
}

type ACLRuleInput struct {
	SubjectKind acl.SubjectKind
	SubjectID   string
	Pattern     string
	Action      acl.Action
}
