package web

import (
	"context"
	"html/template"
	"net/http"

	"github.com/danielzelisko/private-registry/internal/store"
)

var groupsTemplate = template.Must(layoutTemplates.New("groups").Parse(`{{template "head" .}}
{{template "nav" .}}
  <main class="p-6 max-w-4xl">
    <h1 class="text-2xl font-bold text-gray-800">Groups</h1>
    {{if .Error}}<p class="mt-2 text-red-600">{{.Error}}</p>{{end}}
    {{if .Success}}<p class="mt-2 text-green-600">{{.Success}}</p>{{end}}

    <section class="mt-6 bg-white rounded-lg shadow p-4">
      <h2 class="text-lg font-semibold text-gray-800 mb-4">Create group</h2>
      <form method="POST" action="/admin/groups" class="space-y-3">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <div class="flex flex-wrap gap-3 items-end">
          <div>
            <label class="block text-sm text-gray-600 mb-1" for="name">Name</label>
            <input class="border border-gray-300 rounded px-3 py-2" type="text" id="name" name="name" required>
          </div>
          <button class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700" type="submit">Create</button>
        </div>
      </form>
    </section>

    {{range .Groups}}
    <section class="mt-6 bg-white rounded-lg shadow p-4">
      <h2 class="text-lg font-semibold text-gray-800">{{.Name}}</h2>
      <ul class="mt-3 text-sm text-gray-700 space-y-1">
        {{range .Members}}
        <li class="flex items-center gap-3">
          <span>{{.Username}}</span>
          <form method="POST" action="/admin/groups/{{$.GroupID}}/remove-member" class="inline">
            <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
            <input type="hidden" name="user_id" value="{{.ID}}">
            <button class="text-red-600 hover:text-red-800 text-xs" type="submit">Remove</button>
          </form>
        </li>
        {{else}}
        <li class="text-gray-500">No members</li>
        {{end}}
      </ul>
      <form method="POST" action="/admin/groups/{{.GroupID}}/add-member" class="mt-4 flex flex-wrap gap-3 items-end">
        <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
        <div>
          <label class="block text-sm text-gray-600 mb-1">Add member</label>
          <select class="border border-gray-300 rounded px-3 py-2" name="user_id" required>
            {{range $.Users}}
            <option value="{{.ID}}">{{.Username}}</option>
            {{end}}
          </select>
        </div>
        <button class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700" type="submit">Add</button>
      </form>
    </section>
    {{end}}
  </main>
{{template "foot" .}}`))

type groupMember struct {
	ID       string
	Username string
}

type groupView struct {
	GroupID string
	Name    string
	Members []groupMember
}

type groupsPageData struct {
	layoutData
	Users   []store.User
	Groups  []groupView
	Error   string
	Success string
}

func handleGroupsGET(st *store.Store) http.HandlerFunc {
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
		views, err := buildGroupViews(r.Context(), st, groups, users)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := groupsPageData{
			layoutData: layoutData{
				Title:      "Groups",
				Username:   user.Username,
				CSRFToken:  csrfFromContext(r),
				ActivePage: "groups",
			},
			Users:  users,
			Groups: views,
		}
		if msg := r.URL.Query().Get("success"); msg != "" {
			data.Success = msg
		}
		if msg := r.URL.Query().Get("error"); msg != "" {
			data.Error = msg
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = groupsTemplate.Execute(w, data)
	}
}

func handleGroupsPOST(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		name := r.FormValue("name")
		if name == "" {
			http.Redirect(w, r, "/admin/groups?error=Name+required", http.StatusSeeOther)
			return
		}

		if _, err := st.CreateGroup(r.Context(), name); err != nil {
			http.Redirect(w, r, "/admin/groups?error=Could+not+create+group", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/groups?success=Group+created", http.StatusSeeOther)
	}
}

func handleGroupAddMemberPOST(st *store.Store) http.HandlerFunc {
	return groupMemberActionPOST(st, st.AddMember, "Member+added")
}

func handleGroupRemoveMemberPOST(st *store.Store) http.HandlerFunc {
	return groupMemberActionPOST(st, st.RemoveMember, "Member+removed")
}

type groupMemberAction func(ctx context.Context, groupID, userID string) error

func groupMemberActionPOST(st *store.Store, action groupMemberAction, successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !validateCSRF(r) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		groupID := r.PathValue("id")
		userID := r.FormValue("user_id")
		if userID == "" {
			http.Redirect(w, r, "/admin/groups?error=User+required", http.StatusSeeOther)
			return
		}

		if err := action(r.Context(), groupID, userID); err != nil {
			http.Redirect(w, r, "/admin/groups?error=Could+not+update+group", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/groups?success="+successMsg, http.StatusSeeOther)
	}
}

func buildGroupViews(ctx context.Context, st *store.Store, groups []store.Group, users []store.User) ([]groupView, error) {
	membersByGroup := make(map[string][]groupMember)
	for _, u := range users {
		groupIDs, err := st.ListGroupIDsForUser(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		for _, gid := range groupIDs {
			membersByGroup[gid] = append(membersByGroup[gid], groupMember{ID: u.ID, Username: u.Username})
		}
	}

	views := make([]groupView, len(groups))
	for i, g := range groups {
		views[i] = groupView{
			GroupID: g.ID,
			Name:    g.Name,
			Members: membersByGroup[g.ID],
		}
	}
	return views, nil
}
