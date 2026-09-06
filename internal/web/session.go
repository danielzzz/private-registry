package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const sessionCookieName = "session"

type sessionData struct {
	UserID    string
	CSRFToken string
}

type SessionManager struct {
	secret []byte
}

func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{secret: []byte(secret)}
}

func (m *SessionManager) Set(w http.ResponseWriter, userID, csrfToken string) {
	payload := userID + "|" + csrfToken
	sig := m.sign(payload)
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})
}

func (m *SessionManager) Get(r *http.Request) (sessionData, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return sessionData{}, false
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return sessionData{}, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return sessionData{}, false
	}
	payload := string(payloadBytes)
	if !m.verify(payload, parts[1]) {
		return sessionData{}, false
	}
	userID, csrfToken, ok := strings.Cut(payload, "|")
	if !ok || userID == "" || csrfToken == "" {
		return sessionData{}, false
	}
	return sessionData{UserID: userID, CSRFToken: csrfToken}, true
}

func (m *SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (m *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *SessionManager) verify(payload, sig string) bool {
	expected := m.sign(payload)
	return hmac.Equal([]byte(expected), []byte(sig))
}

type contextKey string

const (
	userIDKey    contextKey = "userID"
	csrfTokenKey contextKey = "csrfToken"
)

func userIDFromContext(r *http.Request) (string, bool) {
	v := r.Context().Value(userIDKey)
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok && id != ""
}

var errForbidden = errors.New("forbidden")
