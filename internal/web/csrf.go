package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const csrfFormField = "csrf_token"

func newCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func csrfFromContext(r *http.Request) string {
	v := r.Context().Value(csrfTokenKey)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func withCSRF(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenKey, token)
}

func validateCSRF(r *http.Request) bool {
	expected := csrfFromContext(r)
	if expected == "" {
		return false
	}
	got := r.FormValue(csrfFormField)
	return got != "" && got == expected
}
