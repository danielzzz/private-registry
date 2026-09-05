package acl

type Action string

const (
	ActionPull Action = "pull"
	ActionPush Action = "push"
)

type SubjectKind string

const (
	SubjectUser      SubjectKind = "user"
	SubjectGroup     SubjectKind = "group"
	SubjectAnonymous SubjectKind = "anonymous"
)

type Rule struct {
	SubjectKind SubjectKind
	SubjectID   string
	Pattern     string
	Action      Action
}

type Scope struct {
	Type    string
	Name    string
	Actions []string
}

type Identity struct {
	UserID    string
	GroupIDs  []string
	Anonymous bool
}
