package permission

import "time"

// PublicGroup is the slug for the group whose grants apply to everyone,
// including anonymous users. It is not a membership — its grants are
// always included in every check.
const PublicGroup = "public"

// Valid subject types for tuples.
var validSubjectTypes = map[string]bool{
	"user":  true,
	"group": true,
	"token": true,
}

// Valid object types for tuples.
var validObjectTypes = map[string]bool{
	"site":       true,
	"type":       true,
	"post":       true,
	"page":       true,
	"attachment": true,
	"group":      true,
}

// ValidSubjectType reports whether s is a known subject type.
func ValidSubjectType(s string) bool { return validSubjectTypes[s] }

// ValidObjectType reports whether s is a known object type.
func ValidObjectType(s string) bool { return validObjectTypes[s] }

// Object identifies what is being accessed.
type Object struct {
	Type string // "site", "type", "post", "page", "attachment", "group"
	ID   string // "" for site, "post"/"page"/"attachment" for type, numeric ID for specific
}

// Site returns the site-level object.
func Site() Object { return Object{Type: "site", ID: ""} }

// TypeObject returns an object representing all objects of a post type.
func TypeObject(postType string) Object { return Object{Type: "type", ID: postType} }

// PostObject returns an object for a specific post.
func PostObject(id string) Object { return Object{Type: "post", ID: id} }

// PageObject returns an object for a specific page.
func PageObject(id string) Object { return Object{Type: "page", ID: id} }

// AttachmentObject returns an object for a specific attachment.
func AttachmentObject(id string) Object { return Object{Type: "attachment", ID: id} }

// GroupObject returns an object for a group.
func GroupObject(slug string) Object { return Object{Type: "group", ID: slug} }

// ObjectForPostType returns the correct object for a post type and ID.
// Maps "post" → PostObject, "page" → PageObject, "attachment" → AttachmentObject.
func ObjectForPostType(postType, id string) Object {
	switch postType {
	case "page":
		return PageObject(id)
	case "attachment":
		return AttachmentObject(id)
	default:
		return PostObject(id)
	}
}

// Group represents a permission group (analogous to WordPress roles).
type Group struct {
	ID          int64     `db:"id"`
	Slug        string    `db:"slug"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	IsDefault   bool      `db:"is_default"`
	CreatedAt   time.Time `db:"created_at"`
	CreatedBy   int64     `db:"created_by"`
}

// Tuple represents a subject → relation → object grant.
type Tuple struct {
	ID          int64     `db:"id"`
	SubjectType string    `db:"subject_type"`
	SubjectID   string    `db:"subject_id"`
	Relation    Relation  `db:"relation"`
	ObjectType  string    `db:"object_type"`
	ObjectID    string    `db:"object_id"`
	CreatedAt   time.Time `db:"created_at"`
	CreatedBy   int64     `db:"created_by"`
}

// ShareToken represents a token-based access grant.
type ShareToken struct {
	Token     string     `db:"token"`
	CreatedBy int64      `db:"created_by"`
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt *time.Time `db:"expires_at"`
}
