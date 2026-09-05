package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielzelisko/private-registry/internal/web"
)

func TestHealthz(t *testing.T) {
	h := web.NewHandler(web.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}
