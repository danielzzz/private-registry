package web

import (
	"html/template"
	"net/http"

	"github.com/danielzzz/private-registry/internal/store"
)

var dashboardTemplate = template.Must(layoutTemplates.New("dashboard").Parse(`{{template "head" .}}
{{template "nav" .}}
  <main class="p-6 max-w-4xl">
    <h1 class="text-2xl font-bold text-gray-800">Dashboard</h1>
    <p class="mt-2 text-gray-600">Welcome, {{.Username}}.</p>
    <div class="mt-6 grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="bg-white rounded-lg shadow p-4" data-dashboard-tile="users">
        <p class="text-sm text-gray-500">Users</p>
        <p class="text-2xl font-bold text-gray-800" data-dashboard-count="users">{{.UserCount}}</p>
      </div>
      <div class="bg-white rounded-lg shadow p-4" data-dashboard-tile="groups">
        <p class="text-sm text-gray-500">Groups</p>
        <p class="text-2xl font-bold text-gray-800" data-dashboard-count="groups">{{.GroupCount}}</p>
      </div>
      <div class="bg-white rounded-lg shadow p-4" data-dashboard-tile="tokens">
        <p class="text-sm text-gray-500">API Tokens</p>
        <p class="text-2xl font-bold text-gray-800" data-dashboard-count="tokens">{{.TokenCount}}</p>
      </div>
      <div class="bg-white rounded-lg shadow p-4" data-dashboard-tile="acl">
        <p class="text-sm text-gray-500">ACL Rules</p>
        <p class="text-2xl font-bold text-gray-800" data-dashboard-count="acl">{{.ACLRuleCount}}</p>
      </div>
    </div>
    <p class="mt-6 text-sm text-gray-500">Configure your registry to use this service as its token auth realm.</p>
  </main>
{{template "foot" .}}`))

type dashboardPageData struct {
	layoutData
	UserCount    int
	GroupCount   int
	TokenCount   int
	ACLRuleCount int
}

func handleDashboardGET(st store.Store) http.HandlerFunc {
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

		userCount, err := st.CountUsers(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		groupCount, err := st.CountGroups(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		tokenCount, err := st.CountAPITokens(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		aclCount, err := st.CountACLRules(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = dashboardTemplate.Execute(w, dashboardPageData{
			layoutData: layoutData{
				Title:      "Dashboard",
				Username:   user.Username,
				CSRFToken:  csrfFromContext(r),
				ActivePage: "dashboard",
			},
			UserCount:    userCount,
			GroupCount:   groupCount,
			TokenCount:   tokenCount,
			ACLRuleCount: aclCount,
		})
	}
}
