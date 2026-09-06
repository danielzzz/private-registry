package acl

func Evaluate(id Identity, rules []Rule, requested []Scope) []Scope {
	var out []Scope

	for _, scope := range requested {
		if scope.Type != "repository" {
			continue
		}

		grantedPull, grantedPush := evaluateRules(id, rules, scope.Name)
		actions := grantedActions(grantedPull, grantedPush, scope.Actions)
		if len(actions) == 0 {
			continue
		}

		out = append(out, Scope{
			Type:    scope.Type,
			Name:    scope.Name,
			Actions: actions,
		})
	}

	return out
}

func evaluateRules(id Identity, rules []Rule, name string) (grantedPull, grantedPush bool) {
	for _, rule := range rules {
		if !subjectMatches(id, rule) {
			continue
		}
		if !Match(rule.Pattern, name) {
			continue
		}
		switch rule.Action {
		case ActionPull:
			grantedPull = true
		case ActionPush:
			if id.Anonymous {
				continue
			}
			grantedPull = true
			grantedPush = true
		}
	}
	return grantedPull, grantedPush
}

func subjectMatches(id Identity, rule Rule) bool {
	switch rule.SubjectKind {
	case SubjectAnonymous:
		return id.Anonymous
	case SubjectUser:
		return !id.Anonymous && id.UserID == rule.SubjectID
	case SubjectGroup:
		if id.Anonymous {
			return false
		}
		for _, gid := range id.GroupIDs {
			if gid == rule.SubjectID {
				return true
			}
		}
	}
	return false
}

func grantedActions(grantedPull, grantedPush bool, requested []string) []string {
	requestedSet := make(map[string]struct{}, len(requested))
	for _, a := range requested {
		requestedSet[a] = struct{}{}
	}

	var granted []string
	if grantedPush {
		granted = []string{"pull", "push"}
	} else if grantedPull {
		granted = []string{"pull"}
	}

	var out []string
	for _, a := range granted {
		if _, ok := requestedSet[a]; ok {
			out = append(out, a)
		}
	}
	if containsAction(out, "push") && !containsAction(out, "pull") {
		out = append([]string{"pull"}, out...)
	}
	return out
}

func containsAction(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}
