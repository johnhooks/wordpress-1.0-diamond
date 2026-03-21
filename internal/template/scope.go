package template

// Scope is a chain of variable bindings. Each {each} and {const}
// pushes a new scope. Lookup walks from innermost to outermost,
// then falls through to the page data context.
type Scope struct {
	parent *Scope
	vars   map[string]any
	data   any // page data context (only set on root scope)
}

// NewScope creates a root scope with the given page data context.
func NewScope(data any) *Scope {
	return &Scope{data: data, vars: make(map[string]any)}
}

// Push creates a child scope.
func (s *Scope) Push() *Scope {
	return &Scope{parent: s, vars: make(map[string]any)}
}

// Set binds a variable in this scope.
func (s *Scope) Set(name string, val any) {
	s.vars[name] = val
}

// Lookup resolves an identifier. Checks the scope chain first,
// then the page data context. Returns the value and the context
// to use for further member access.
//
// For a bare identifier like "post", if "post" is a scope variable,
// it returns (postValue, postValue). If not, it tries the page data
// as a struct field lookup and returns (fieldValue, fieldValue).
//
// This means {post.Title} first resolves "post" in scope, then
// resolves "Title" on the result.
func (s *Scope) Lookup(name string) (any, bool) {
	// Walk the scope chain.
	for cur := s; cur != nil; cur = cur.parent {
		if val, ok := cur.vars[name]; ok {
			return val, true
		}
	}
	// Fall through to page data context.
	root := s.root()
	if root.data == nil {
		return nil, false
	}
	val, err := lookupField(root.data, name)
	if err != nil {
		return nil, false
	}
	return val, true
}

// Data returns the root page data context.
func (s *Scope) Data() any {
	return s.root().data
}

func (s *Scope) root() *Scope {
	cur := s
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur
}
