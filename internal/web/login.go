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

var adminTemplate = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Admin</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
  <nav class="bg-white shadow px-6 py-4 flex justify-between items-center">
    <span class="font-semibold text-gray-800">Private Registry Admin</span>
    <form method="POST" action="/logout">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <button class="text-sm text-gray-600 hover:text-gray-900" type="submit">Logout</button>
    </form>
  </nav>
  <main class="p-6">
    <h1 class="text-2xl font-bold text-gray-800">Dashboard</h1>
    <p class="mt-2 text-gray-600">Welcome, {{.Username}}.</p>
  </main>
</body>
</html>`))

type loginPageData struct {
	Error string
}

type adminPageData struct {
	Username  string
	CSRFToken string
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
