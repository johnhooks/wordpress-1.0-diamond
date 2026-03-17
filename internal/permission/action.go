package permission

// Action represents what the caller wants to do. The permission system
// maps actions to required relations.
type Action string

const (
	ActionView             Action = "view"
	ActionComment          Action = "comment"
	ActionCreate           Action = "create"
	ActionEdit             Action = "edit"
	ActionDelete           Action = "delete"
	ActionPublish          Action = "publish"
	ActionUpload           Action = "upload"
	ActionModerate         Action = "moderate"
	ActionManageCategories Action = "manage_categories"
	ActionManageLinks      Action = "manage_links"
	ActionManageOptions    Action = "manage_options"
	ActionManageUsers      Action = "manage_users"
)

// actionRelation maps each action to the minimum relation required.
var actionRelation = map[Action]Relation{
	ActionView:             Viewer,
	ActionComment:          Commenter,
	ActionCreate:           Writer,
	ActionEdit:             Writer,
	ActionDelete:           Writer,
	ActionPublish:          Writer,
	ActionUpload:           Writer,
	ActionModerate:         Moderator,
	ActionManageCategories: Moderator,
	ActionManageLinks:      Moderator,
	ActionManageOptions:    Owner,
	ActionManageUsers:      Owner,
}

// RequiredRelation returns the minimum relation needed for an action.
func RequiredRelation(action Action) (Relation, bool) {
	r, ok := actionRelation[action]
	return r, ok
}

// ownershipActions are actions that a type-level writer grant does NOT
// cover. These require either an editor grant at the type level (which
// covers any object) or a direct grant on the specific object (e.g.,
// the authorship tuple created when a post is written).
var ownershipActions = map[Action]bool{
	ActionEdit:    true,
	ActionDelete:  true,
	ActionPublish: true,
}
