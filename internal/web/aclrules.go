package web

import (
	"html/template"
	"net/http"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/danielzelisko/private-registry/internal/store"
)

var aclRulesTemplate = template.Must(layoutTemplates.New("aclrules").Parse(`{{template "head" .}}
{{template "nav" .}}
  <main class="p-6 max-w-4xl">
    <h1 class="text-2xl font-bold text-gray-800">ACL Rules</h1>
    {{if .Error}}<p class="mt-2 text-red-600">{{.Error}}</p>{{end}}
    {{if .Success}}<p class="mt-2 text-green-600">{{.Success}}</p>{{end}}

    <section class="mt-6 bg-white rounded-lg shadow p-4">
      <h2 class="text-lg font-semibold text-gray-800 mb-4">Create rule</h2>
      <form method="POST" action="/admin/acl" class="space-y-3">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <div class="flex flex-wrap gap-3 items-end">
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="subject_kind">Subject</label>
            <select class="border border-gray-300 rounded px-3 py-2" id="subject_kind" name="subject_kind" required>
              <option value="user">User</option>
              <option value="group">Group</option>
              <option value="anonymous">Anonymous</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="subject_id">Subject ID</label>
            <select class="border border-gray-300 rounded px-3 py-2" id="subject_id" name="subject_id">
              <option value="">(none)</option>
              {{range .Users}}
              <option value="{{.ID}}">user: {{.Username}}</option>
              {{end}}
              {{range .Groups}}
              <option value="{{.ID}}">group: {{.Name}}</option>
              {{end}}
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="pattern">Pattern</label>
            <input class="border border-gray-300 rounded px-3 py-2" type="text" id="pattern" name="pattern" required placeholder="app/*">
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="action">Action</label>
            <select class="border border-gray-300 rounded px-3 py-2" id="action" name="action" required>
              <option value="pull">pull</option>
              <option value="push" id="action-push">push</option>
            </select>
          </div>
          <button class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700" type="submit">Create</button>
        </div>
      </form>
    </section>

    <section class="mt-6 bg-white rounded-lg shadow overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-left text-gray-600">
          <tr>
            <th class="px-4 py-3">Subject</th>
            <th class="px-4 py-3">Pattern</th>
            <th class="px-4 py-3">Action</th>
            <th class="px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          {{range .Rules}}
          <tr class="border-t border-gray-100">
            <td class="px-4 py-3">{{.SubjectLabel}}</td>
            <td class="px-4 py-3 font-mono text-gray-800">{{.Pattern}}</td>
            <td class="px-4 py-3">{{.Action}}</td>
            <td class="px-4 py-3">
              <form method="POST" action="/admin/acl/{{.ID}}/delete" class="inline">
                <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                <button class="text-red-600 hover:text-red-800" type="submit">Delete</button>
              </form>
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </section>
  </main>
  <script>
    (function () {
      var subjectKind = document.getElementById('subject_kind');
      var action = document.getElementById('action');
      var pushOption = document.getElementById('action-push');
      function syncActionOptions() {
        var anonymous = subjectKind.value === 'anonymous';
        pushOption.hidden = anonymous;
        pushOption.disabled = anonymous;
        if (anonymous && action.value === 'push') {
          action.value = 'pull';
        }
      }
      subjectKind.addEventListener('change', syncActionOptions);
      syncActionOptions();
    })();
  </script>
{{template "foot" .}}`))

type aclRuleRow struct {
	ID           string
	SubjectLabel string
	Pattern      string
	Action       string
}

type aclRulesPageData struct {
	layoutData
	Users   []store.User
	Groups  []store.Group
	Rules   []aclRuleRow
	Error   string
	Success string
}

func handleACLRulesGET(st *store.Store) http.HandlerFunc {
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
		groups, err := st.ListGroups(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		rules, err := st.ListACLRules(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := aclRulesPageData{
			layoutData: layoutData{
				Title:      "ACL Rules",
				Username:   user.Username,
				CSRFToken:  csrfFromContext(r),
				ActivePage: "acl",
			},
			Users:  users,
			Groups: groups,
			Rules:  buildACLRuleRows(rules, users, groups),
		}
		if msg := r.URL.Query().Get("success"); msg != "" {
			data.Success = msg
		}
		if msg := r.URL.Query().Get("error"); msg != "" {
			data.Error = msg
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = aclRulesTemplate.Execute(w, data)
	}
}

func handleACLRulesPOST(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		subjectKind := acl.SubjectKind(r.FormValue("subject_kind"))
		subjectID := r.FormValue("subject_id")
		pattern := r.FormValue("pattern")
		action := acl.Action(r.FormValue("action"))

		if pattern == "" || action == "" {
			http.Redirect(w, r, "/admin/acl?error=Pattern+and+action+required", http.StatusSeeOther)
			return
		}
		if subjectKind == acl.SubjectAnonymous {
			subjectID = ""
			if action == acl.ActionPush {
				http.Redirect(w, r, "/admin/acl?error=Anonymous+push+is+not+allowed", http.StatusSeeOther)
				return
			}
		} else if subjectID == "" {
			http.Redirect(w, r, "/admin/acl?error=Subject+required", http.StatusSeeOther)
			return
		}

		_, err := st.CreateACLRule(r.Context(), store.ACLRuleInput{
			SubjectKind: subjectKind,
			SubjectID:   subjectID,
			Pattern:     pattern,
			Action:      action,
		})
		if err != nil {
			http.Redirect(w, r, "/admin/acl?error=Could+not+create+rule", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/acl?success=Rule+created", http.StatusSeeOther)
	}
}

func handleACLRuleDeletePOST(st *store.Store) http.HandlerFunc {
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
		if err := st.DeleteACLRule(r.Context(), id); err != nil {
			http.Redirect(w, r, "/admin/acl?error=Could+not+delete+rule", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/acl?success=Rule+deleted", http.StatusSeeOther)
	}
}

func buildACLRuleRows(rules []store.ACLRule, users []store.User, groups []store.Group) []aclRuleRow {
	userNames := make(map[string]string, len(users))
	for _, u := range users {
		userNames[u.ID] = u.Username
	}
	groupNames := make(map[string]string, len(groups))
	for _, g := range groups {
		groupNames[g.ID] = g.Name
	}

	rows := make([]aclRuleRow, len(rules))
	for i, rule := range rules {
		label := subjectLabel(rule, userNames, groupNames)
		rows[i] = aclRuleRow{
			ID:           rule.ID,
			SubjectLabel: label,
			Pattern:      rule.Pattern,
			Action:       string(rule.Action),
		}
	}
	return rows
}

func subjectLabel(rule store.ACLRule, userNames, groupNames map[string]string) string {
	switch rule.SubjectKind {
	case acl.SubjectAnonymous:
		return "anonymous"
	case acl.SubjectUser:
		if name, ok := userNames[rule.SubjectID]; ok {
			return "user: " + name
		}
		return "user: " + rule.SubjectID
	case acl.SubjectGroup:
		if name, ok := groupNames[rule.SubjectID]; ok {
			return "group: " + name
		}
		return "group: " + rule.SubjectID
	default:
		return string(rule.SubjectKind)
	}
}
