package web

import "html/template"

var layoutTemplates = template.Must(template.New("layout").Parse(`{{define "head"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
{{end}}
{{define "nav"}}
  <nav class="bg-white shadow px-6 py-4 flex justify-between items-center">
    <div class="flex items-center gap-6">
      <span class="font-semibold text-gray-800">Private Registry Admin</span>
      <a class="text-sm {{if eq .ActivePage "dashboard"}}text-blue-600 font-medium{{else}}text-gray-600 hover:text-gray-900{{end}}" href="/admin/">Dashboard</a>
      <a class="text-sm {{if eq .ActivePage "users"}}text-blue-600 font-medium{{else}}text-gray-600 hover:text-gray-900{{end}}" href="/admin/users">Users</a>
      <a class="text-sm {{if eq .ActivePage "tokens"}}text-blue-600 font-medium{{else}}text-gray-600 hover:text-gray-900{{end}}" href="/admin/tokens">API Tokens</a>
      <a class="text-sm {{if eq .ActivePage "groups"}}text-blue-600 font-medium{{else}}text-gray-600 hover:text-gray-900{{end}}" href="/admin/groups">Groups</a>
      <a class="text-sm {{if eq .ActivePage "acl"}}text-blue-600 font-medium{{else}}text-gray-600 hover:text-gray-900{{end}}" href="/admin/acl">ACL Rules</a>
    </div>
    <form method="POST" action="/logout">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <button class="text-sm text-gray-600 hover:text-gray-900" type="submit">Logout</button>
    </form>
  </nav>
{{end}}
{{define "foot"}}
</body>
</html>
{{end}}`))

type layoutData struct {
	Title      string
	Username   string
	CSRFToken  string
	ActivePage string
}
