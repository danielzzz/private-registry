package web

import (
	"html/template"
	"net/http"

	"github.com/danielzelisko/private-registry/internal/auth"
)

var loginTemplate = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Login</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center">
  <div class="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
    <h1 class="text-2xl font-bold mb-6 text-gray-800">Admin Login</h1>
    {{if .Error}}<p class="text-red-600 mb-4">{{.Error}}</p>{{end}}
    <form method="POST" action="/login" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="username">Username</label>
        <input class="w-full border border-gray-300 rounded px-3 py-2" type="text" id="username" name="username" required autofocus>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="password">Password</label>
        <input class="w-full border border-gray-300 rounded px-3 py-2" type="password" id="password" name="password" required>
      </div>
      <button class="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700" type="submit">Sign in</button>
    </form>
  </div>
</body>
</html>`))

type loginPageData struct {
	Error string
}

func handleLoginGET(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = loginTemplate.Execute(w, loginPageData{})
}

func handleLoginPOST(sessions *SessionManager, authn *auth.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password")

		user, err := authn.Authenticate(r.Context(), username, password)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = loginTemplate.Execute(w, loginPageData{Error: "Invalid username or password"})
			return
		}
		if !user.Admin {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = loginTemplate.Execute(w, loginPageData{Error: "Admin access required. Registry users should use docker login, not this page."})
			return
		}

		csrf, err := newCSRFToken()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		sessions.Set(w, user.ID, csrf)
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	}
}

func handleLogout(sessions *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}
		sessions.Clear(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
