package acl_test

import (
	"testing"

	"github.com/danielzzz/private-registry/internal/acl"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"myorg/test-project*", "myorg/test-project", true},
		{"myorg/test-project*", "myorg/test-project-app", true},
		{"myorg/test-project*", "myorg/other", false},
		{"library/*", "library/nginx", true},
		{"library/*", "library/nginx/extra", false},
		{"*", "thing", true},
		{"*", "any/thing", false},
		{"*/*", "any/thing", true},
		{"exact/name", "exact/name", true},
		{"exact/name", "exact/other", false},
	}
	for _, tc := range cases {
		got := acl.Match(tc.pattern, tc.name)
		if got != tc.want {
			t.Fatalf("Match(%q,%q)=%v want %v", tc.pattern, tc.name, got, tc.want)
		}
	}
}
