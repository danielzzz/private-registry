package web

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/danielzzz/private-registry/internal/auth"
	"github.com/danielzzz/private-registry/internal/store"
)

var usersTemplate = template.Must(layoutTemplates.New("users").Parse(`{{template "head" .}}
{{template "nav" .}}
  <main class="p-6 max-w-4xl">
    <h1 class="text-2xl font-bold text-gray-800">Users</h1>
    {{if .Error}}<p class="mt-2 text-red-600">{{.Error}}</p>{{end}}
    {{if .Success}}<p class="mt-2 text-green-600">{{.Success}}</p>{{end}}

    <section class="mt-6 bg-white rounded-lg shadow p-4">
      <h2 class="text-lg font-semibold text-gray-800 mb-4">Create user</h2>
      <form method="POST" action="/admin/users" class="space-y-3">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <div class="flex flex-wrap gap-3 items-end">
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="username">Username</label>
            <input class="border border-gray-300 rounded px-3 py-2" type="text" id="username" name="username" required>
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="password">Password</label>
            <input class="border border-gray-300 rounded px-3 py-2" type="password" id="password" name="password" required>
          </div>
          <label class="flex items-center gap-2 text-sm text-gray-700">
            <input type="checkbox" name="admin" value="1"> Admin
          </label>
          <button class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700" type="submit">Create</button>
        </div>
      </form>
    </section>

    <section class="mt-6 bg-white rounded-lg shadow overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-left text-gray-600">
          <tr>
            <th class="px-4 py-3">Username</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3">Admin</th>
            <th class="px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          {{range .Users}}
          <tr class="border-t border-gray-100">
            <td class="px-4 py-3 font-medium text-gray-800">{{.Username}}</td>
            <td class="px-4 py-3">{{if .Active}}Active{{else}}Disabled{{end}}</td>
            <td class="px-4 py-3">{{if .Admin}}Yes{{else}}No{{end}}</td>
            <td class="px-4 py-3">
              <div class="flex flex-wrap gap-2 items-center">
                {{if .Active}}
                <form method="POST" action="/admin/users/{{.ID}}/disable" class="inline">
                  <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                  <button class="text-red-600 hover:text-red-800" type="submit">Disable</button>
                </form>
                {{else}}
                <form method="POST" action="/admin/users/{{.ID}}/enable" class="inline">
                  <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                  <button class="text-green-600 hover:text-green-800" type="submit">Enable</button>
                </form>
                {{end}}
                <form method="POST" action="/admin/users/{{.ID}}/toggle-admin" class="inline">
                  <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                  <button class="text-gray-600 hover:text-gray-900" type="submit">{{if .Admin}}Remove admin{{else}}Make admin{{end}}</button>
                </form>
                <form method="POST" action="/admin/users/{{.ID}}/reset-password" class="inline flex items-center gap-1">
                  <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                  <input class="border border-gray-300 rounded px-2 py-1 text-xs" type="password" name="password" placeholder="New password" required>
                  <button class="text-gray-600 hover:text-gray-900" type="submit">Reset</button>
                </form>
              </div>
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </section>
  </main>
{{template "foot" .}}`))

type usersPageData struct {
	layoutData
	Users   []store.User
	Error   string
	Success string
}

func handleUsersGET(st store.Store) http.HandlerFunc {
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

		data := usersPageData{
			layoutData: layoutData{
				Title:      "Users",
				Username:   user.Username,
				CSRFToken:  csrfFromContext(r),
				ActivePage: "users",
			},
			Users: users,
		}
		if msg := r.URL.Query().Get("success"); msg != "" {
			data.Success = msg
		}
		if msg := r.URL.Query().Get("error"); msg != "" {
			data.Error = msg
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = usersTemplate.Execute(w, data)
	}
}

func handleUsersPOST(st store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		admin := r.FormValue("admin") == "1"

		if username == "" || password == "" {
			http.Redirect(w, r, "/admin/users?error=Username+and+password+required", http.StatusSeeOther)
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		_, err = st.CreateUser(r.Context(), username, hash, admin)
		if err != nil {
			http.Redirect(w, r, "/admin/users?error=Could+not+create+user", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/admin/users?success=User+created", http.StatusSeeOther)
	}
}

func handleUserDisablePOST(st store.Store) http.HandlerFunc {
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
		if err := guardLastActiveAdmin(r.Context(), st, id); err != nil {
			http.Redirect(w, r, "/admin/users?error=Cannot+disable+the+last+active+admin", http.StatusSeeOther)
			return
		}
		if err := st.SetUserActive(r.Context(), id, false); err != nil {
			http.Redirect(w, r, "/admin/users?error=Could+not+update+user", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/users?success=User+disabled", http.StatusSeeOther)
	}
}

func handleUserEnablePOST(st store.Store) http.HandlerFunc {
	return userActionPOST(st, func(ctx context.Context, st store.Store, id string) error {
		return st.SetUserActive(ctx, id, true)
	}, "User+enabled")
}

func handleUserToggleAdminPOST(st store.Store) http.HandlerFunc {
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
		u, err := st.GetUserByID(r.Context(), id)
		if err != nil {
			http.Redirect(w, r, "/admin/users?error=User+not+found", http.StatusSeeOther)
			return
		}

		if u.Admin {
			if err := guardLastActiveAdmin(r.Context(), st, id); err != nil {
				http.Redirect(w, r, "/admin/users?error=Cannot+remove+the+last+active+admin", http.StatusSeeOther)
				return
			}
		}

		if err := st.SetUserAdmin(r.Context(), id, !u.Admin); err != nil {
			http.Redirect(w, r, "/admin/users?error=Could+not+update+user", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/users?success=User+updated", http.StatusSeeOther)
	}
}

func handleUserResetPasswordPOST(st store.Store) http.HandlerFunc {
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
		password := r.FormValue("password")
		if password == "" {
			http.Redirect(w, r, "/admin/users?error=Password+required", http.StatusSeeOther)
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := st.UpdatePasswordHash(r.Context(), id, hash); err != nil {
			http.Redirect(w, r, "/admin/users?error=Could+not+reset+password", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/users?success=Password+reset", http.StatusSeeOther)
	}
}

// guardLastActiveAdmin returns an error when the target is the only active admin.
func guardLastActiveAdmin(ctx context.Context, st store.Store, targetID string) error {
	u, err := st.GetUserByID(ctx, targetID)
	if err != nil {
		return err
	}
	if !u.Active || !u.Admin {
		return nil
	}
	n, err := st.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return fmt.Errorf("last active admin")
	}
	return nil
}

type userAction func(ctx context.Context, st store.Store, id string) error

func userActionPOST(st store.Store, action userAction, successMsg string) http.HandlerFunc {
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
		if err := action(r.Context(), st, id); err != nil {
			http.Redirect(w, r, "/admin/users?error=Could+not+update+user", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/users?success="+successMsg, http.StatusSeeOther)
	}
}
