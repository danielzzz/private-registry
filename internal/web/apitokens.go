package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
)

var apiTokensTemplate = template.Must(layoutTemplates.New("apitokens").Parse(`{{template "head" .}}
{{template "nav" .}}
  <main class="p-6 max-w-4xl">
    <h1 class="text-2xl font-bold text-gray-800">API Tokens</h1>
    {{if .Error}}<p class="mt-2 text-red-600">{{.Error}}</p>{{end}}
    {{if .Success}}<p class="mt-2 text-green-600">{{.Success}}</p>{{end}}
    {{if .CreatedToken}}
    <div class="mt-4 bg-yellow-50 border border-yellow-200 rounded-lg p-4">
      <p class="text-sm font-medium text-yellow-800">Copy this token now. It will not be shown again.</p>
      <code class="mt-2 block text-sm break-all text-gray-900">{{.CreatedToken}}</code>
    </div>
    {{end}}

    <section class="mt-6 bg-white rounded-lg shadow p-4">
      <h2 class="text-lg font-semibold text-gray-800 mb-4">Create token</h2>
      <form method="POST" action="/admin/tokens" class="space-y-3">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <div class="flex flex-wrap gap-3 items-end">
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="user_id">User</label>
            <select class="border border-gray-300 rounded px-3 py-2" id="user_id" name="user_id" required>
              {{range .Users}}
              <option value="{{.ID}}">{{.Username}}</option>
              {{end}}
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="name">Name</label>
            <input class="border border-gray-300 rounded px-3 py-2" type="text" id="name" name="name" required>
          </div>
          <button class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700" type="submit">Create</button>
        </div>
      </form>
    </section>

    <section class="mt-6 bg-white rounded-lg shadow overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-left text-gray-600">
          <tr>
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">User</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          {{range .Tokens}}
          <tr class="border-t border-gray-100">
            <td class="px-4 py-3 font-medium text-gray-800">{{.Name}}</td>
            <td class="px-4 py-3">{{.Username}}</td>
            <td class="px-4 py-3">{{if .Active}}Active{{else}}Revoked{{end}}</td>
            <td class="px-4 py-3">
              {{if .Active}}
              <form method="POST" action="/admin/tokens/{{.ID}}/revoke" class="inline">
                <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                <button class="text-red-600 hover:text-red-800" type="submit">Revoke</button>
              </form>
              {{end}}
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </section>
  </main>
{{template "foot" .}}`))

type tokenRow struct {
	ID       string
	Name     string
	Username string
	Active   bool
}

type apiTokensPageData struct {
	layoutData
	Users        []store.User
	Tokens       []tokenRow
	Error        string
	Success      string
	CreatedToken string
}

func handleAPITokensGET(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := st.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		users, err := st.ListUsers(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		tokens, err := listAllTokens(r.Context(), st, users)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := apiTokensPageData{
			layoutData: layoutData{
				Title:      "API Tokens",
				Username:   user.Username,
				CSRFToken:  csrfFromContext(r),
				ActivePage: "tokens",
			},
			Users:  users,
			Tokens: tokens,
		}
		if msg := r.URL.Query().Get("success"); msg != "" {
			data.Success = msg
		}
		if msg := r.URL.Query().Get("error"); msg != "" {
			data.Error = msg
		}
		if created := r.URL.Query().Get("created"); created != "" {
			data.CreatedToken = created
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = apiTokensTemplate.Execute(w, data)
	}
}

func handleAPITokensPOST(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		userID := r.FormValue("user_id")
		name := r.FormValue("name")
		if userID == "" || name == "" {
			http.Redirect(w, r, "/admin/tokens?error=User+and+name+required", http.StatusSeeOther)
			return
		}

		secret, err := generateTokenSecret()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		hash, err := auth.HashPassword(secret)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		tokenID, err := st.CreateAPIToken(r.Context(), userID, name, hash, nil)
		if err != nil {
			http.Redirect(w, r, "/admin/tokens?error=Could+not+create+token", http.StatusSeeOther)
			return
		}

		plaintext := fmt.Sprintf("prt_%s_%s", tokenID, secret)
		http.Redirect(w, r, "/admin/tokens?created="+url.QueryEscape(plaintext), http.StatusSeeOther)
	}
}

func handleAPITokenRevokePOST(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		id := r.PathValue("id")
		if err := st.RevokeAPIToken(r.Context(), id); err != nil {
			http.Redirect(w, r, "/admin/tokens?error=Could+not+revoke+token", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/tokens?success=Token+revoked", http.StatusSeeOther)
	}
}

func listAllTokens(ctx context.Context, st *store.Store, users []store.User) ([]tokenRow, error) {
	usernames := make(map[string]string, len(users))
	for _, u := range users {
		usernames[u.ID] = u.Username
	}

	var rows []tokenRow
	for _, u := range users {
		tokens, err := st.ListAPITokensByUser(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		for _, t := range tokens {
			rows = append(rows, tokenRow{
				ID:       t.ID,
				Name:     t.Name,
				Username: usernames[t.UserID],
				Active:   t.Active,
			})
		}
	}
	return rows, nil
}

func generateTokenSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
