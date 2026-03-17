package permission

// Relation represents the relationship between a subject and an object.
// Relations are hierarchical: a higher relation implies all lower ones
// in its branch.
type Relation string

const (
	Member    Relation = "member"
	Viewer    Relation = "viewer"
	Commenter Relation = "commenter"
	Writer    Relation = "writer"
	Editor    Relation = "editor"
	Moderator Relation = "moderator"
	Owner     Relation = "owner"
)

// implies maps each relation to the relations it transitively grants.
// Owner implies everything except Member (which is a separate branch).
var implies = map[Relation][]Relation{
	Owner:     {Editor, Moderator, Writer, Commenter, Viewer},
	Editor:    {Writer, Commenter, Viewer},
	Writer:    {Commenter, Viewer},
	Commenter: {Viewer},
	Viewer:    {},
	Moderator: {},
	Member:    {},
}

// ValidRelation reports whether r is a known relation.
// Derived from the implies map — no separate list to keep in sync.
func ValidRelation(r Relation) bool {
	_, ok := implies[r]
	return ok
}

// Satisfies returns true if held implies or equals required.
func (held Relation) Satisfies(required Relation) bool {
	if held == required {
		return true
	}
	for _, r := range implies[held] {
		if r == required {
			return true
		}
	}
	return false
}
