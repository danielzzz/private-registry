// Command demoseed creates local-demo users and ACL rules in the auth SQLite DB.
// Intended for examples/local-demo only; not part of the production server image.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/danielzzz/private-registry/internal/acl"
	"github.com/danielzzz/private-registry/internal/auth"
	"github.com/danielzzz/private-registry/internal/store"
	"github.com/danielzzz/private-registry/internal/store/sqlite"
)

const (
	defaultDBPath = "/data/registry-auth.db"
	repoPattern   = "private-registry/*"
)

type demoUser struct {
	username string
	password string
	action   acl.Action
}

var demoUsers = []demoUser{
	{username: "writer", password: "writerpass", action: acl.ActionPush},
	{username: "reader", password: "readerpass", action: acl.ActionPull},
}

func main() {
	dbPath := flag.String("db", defaultDBPath, "path to auth SQLite database")
	flag.Parse()

	if err := run(*dbPath); err != nil {
		log.Fatalf("demoseed: %v", err)
	}
}

func run(dbPath string) error {
	s, err := sqlite.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := context.Background()
	rules, err := s.ListACLRules(ctx)
	if err != nil {
		return fmt.Errorf("list acl: %w", err)
	}

	for _, du := range demoUsers {
		user, created, err := ensureUser(ctx, s, du.username, du.password)
		if err != nil {
			return err
		}
		if created {
			fmt.Printf("created user %s\n", du.username)
		} else {
			fmt.Printf("user %s already exists\n", du.username)
		}

		if hasRule(rules, user.ID, repoPattern, du.action) {
			fmt.Printf("acl %s %s %s already exists\n", du.username, repoPattern, du.action)
			continue
		}
		_, err = s.CreateACLRule(ctx, store.ACLRuleInput{
			SubjectKind: acl.SubjectUser,
			SubjectID:   user.ID,
			Pattern:     repoPattern,
			Action:      du.action,
		})
		if err != nil {
			return fmt.Errorf("create acl for %s: %w", du.username, err)
		}
		fmt.Printf("created acl %s %s %s\n", du.username, repoPattern, du.action)
		rules = append(rules, store.ACLRule{
			SubjectKind: acl.SubjectUser,
			SubjectID:   user.ID,
			Pattern:     repoPattern,
			Action:      du.action,
		})
	}

	return nil
}

func ensureUser(ctx context.Context, s store.Store, username, password string) (store.User, bool, error) {
	existing, err := s.GetUserByUsername(ctx, username)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.User{}, false, fmt.Errorf("get user %s: %w", username, err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return store.User{}, false, fmt.Errorf("hash password for %s: %w", username, err)
	}
	user, err := s.CreateUser(ctx, username, hash, false)
	if err != nil {
		return store.User{}, false, fmt.Errorf("create user %s: %w", username, err)
	}
	return user, true, nil
}

func hasRule(rules []store.ACLRule, subjectID, pattern string, action acl.Action) bool {
	for _, r := range rules {
		if r.SubjectKind == acl.SubjectUser &&
			r.SubjectID == subjectID &&
			r.Pattern == pattern &&
			r.Action == action {
			return true
		}
	}
	return false
}

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
}
