package auth_test

import (
	"testing"

	"github.com/danielzzz/private-registry/internal/auth"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !auth.CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to verify")
	}
}

func TestCheckPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPasswordUsesUniqueSalts(t *testing.T) {
	h1, err := auth.HashPassword("same")
	if err != nil {
		t.Fatal(err)
	}
	h2, err := auth.HashPassword("same")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatal("expected different hashes for same password")
	}
}
