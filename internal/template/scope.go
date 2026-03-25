package template

import "reflect"

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

// PushData creates a child scope with a view-tagged struct's fields
// flattened as variables. If data is nil, returns a plain child scope.
// Fields tagged `view:"name,raw"` are wrapped in RawHTML. Parent scope
// variables remain accessible through the scope chain.
func (s *Scope) PushData(data any) *Scope {
	child := s.Push()
	if data == nil {
		return child
	}

	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return child
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return child
	}

	fields := viewFields(v.Type())
	for name, info := range fields {
		val := v.FieldByIndex(info.Index).Interface()
		if info.Raw {
			if s, ok := val.(string); ok {
				val = RawHTML(s)
			}
		}
		child.Set(name, val)
	}
	return child
}

func (s *Scope) root() *Scope {
	cur := s
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur
}
