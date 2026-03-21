package template

import (
	"fmt"
	"html"
	"reflect"
	"strings"

	"press/internal/template/parse"
)

// Eval evaluates an expression against a data context and returns
// the resulting value. The context can be a *Scope (which checks
// bindings first then page data) or any struct/map value.
func Eval(expr parse.Expr, ctx any) (any, error) {
	switch e := expr.(type) {
	case *parse.Ident:
		return lookupIdent(ctx, e.Name)
	case *parse.MemberExpr:
		obj, err := Eval(e.Object, ctx)
		if err != nil {
			return nil, err
		}
		return lookupField(obj, e.Field)
	case *parse.StringLit:
		return e.Value, nil
	case *parse.NumberLit:
		return e.Value, nil
	case *parse.BoolLit:
		return e.Value, nil
	case *parse.BinaryExpr:
		return evalBinary(e, ctx)
	case *parse.UnaryExpr:
		return evalUnary(e, ctx)
	default:
		return nil, fmt.Errorf("unknown expression type %T", expr)
	}
}

// EvalString evaluates an expression and formats the result as a
// string suitable for insertion into HTML text content.
func EvalString(expr parse.Expr, ctx any) (string, error) {
	val, err := Eval(expr, ctx)
	if err != nil {
		return "", err
	}
	return formatValue(val), nil
}

// EvalTruthy evaluates an expression and returns its truthiness.
//
// Falsy: empty string, 0, false, nil, empty slice, empty map.
// Truthy: everything else.
func EvalTruthy(expr parse.Expr, ctx any) (bool, error) {
	val, err := Eval(expr, ctx)
	if err != nil {
		return false, err
	}
	return IsTruthy(val), nil
}

// IsTruthy returns the truthiness of a value.
func IsTruthy(val any) bool {
	if val == nil {
		return false
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.String:
		return v.String() != ""
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.Slice, reflect.Map:
		return v.Len() > 0
	case reflect.Ptr, reflect.Interface:
		return !v.IsNil()
	default:
		return true
	}
}

// EvalAttrParts evaluates pre-compiled attribute parts against a
// context and returns the assembled string. Expression values are
// HTML-escaped; static text passes through unchanged.
func EvalAttrParts(parts []parse.AttrPart, ctx any) (string, error) {
	var sb strings.Builder
	for _, part := range parts {
		if part.Expr != nil {
			val, err := EvalString(part.Expr, ctx)
			if err != nil {
				return "", err
			}
			sb.WriteString(html.EscapeString(val))
		} else {
			sb.WriteString(part.Text)
		}
	}
	return sb.String(), nil
}

// lookupIdent resolves a bare identifier. If the context is a *Scope,
// it checks scope bindings first, then falls through to the page data.
// Otherwise it does a direct field lookup.
func lookupIdent(ctx any, name string) (any, error) {
	if scope, ok := ctx.(*Scope); ok {
		val, found := scope.Lookup(name)
		if found {
			return val, nil
		}
		return nil, fmt.Errorf("undefined variable %q", name)
	}
	return lookupField(ctx, name)
}

// lookupField resolves a field name on a value using reflection.
// Supports struct fields and map string keys. Methods are not
// callable from templates. If the engine needs a computed value,
// it belongs in the view struct as a field.
func lookupField(ctx any, name string) (any, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cannot access %q on nil", name)
	}

	v := reflect.ValueOf(ctx)

	// Unwrap pointers and interfaces.
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		field := v.FieldByName(name)
		if field.IsValid() {
			return field.Interface(), nil
		}
		return nil, fmt.Errorf("no field %q on %T", name, ctx)

	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			result := v.MapIndex(reflect.ValueOf(name))
			if result.IsValid() {
				return result.Interface(), nil
			}
			return nil, nil
		}
		return nil, fmt.Errorf("map key type is %s, not string", v.Type().Key())

	default:
		return nil, fmt.Errorf("cannot access %q on %s", name, v.Kind())
	}
}

func evalBinary(e *parse.BinaryExpr, ctx any) (any, error) {
	// Logical operators short-circuit: evaluate the right side only
	// when the left side does not determine the result. Both return
	// the deciding value, not a boolean, matching JavaScript/Svelte
	// semantics. This makes {if obj and obj.Field} safe when obj is
	// nil, and {name or "Anonymous"} return the string.
	switch e.Op {
	case "and":
		left, err := Eval(e.Left, ctx)
		if err != nil {
			return nil, err
		}
		if !IsTruthy(left) {
			return left, nil
		}
		return Eval(e.Right, ctx)
	case "or":
		left, err := Eval(e.Left, ctx)
		if err != nil {
			return nil, err
		}
		if IsTruthy(left) {
			return left, nil
		}
		return Eval(e.Right, ctx)
	}

	// All other operators need both sides evaluated.
	left, err := Eval(e.Left, ctx)
	if err != nil {
		return nil, err
	}
	right, err := Eval(e.Right, ctx)
	if err != nil {
		return nil, err
	}

	switch e.Op {
	case "==":
		return isEqual(left, right), nil
	case "!=":
		return !isEqual(left, right), nil
	case "<", ">", "<=", ">=":
		return compareOrdered(left, right, e.Op)
	default:
		return nil, fmt.Errorf("unknown operator %q", e.Op)
	}
}

func evalUnary(e *parse.UnaryExpr, ctx any) (any, error) {
	val, err := Eval(e.Operand, ctx)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case "not":
		return !IsTruthy(val), nil
	default:
		return nil, fmt.Errorf("unknown unary operator %q", e.Op)
	}
}

func isEqual(left, right any) bool {
	// Try numeric comparison first to handle int vs float64.
	lf, lok := toFloat(left)
	rf, rok := toFloat(right)
	if lok && rok {
		return lf == rf
	}
	return reflect.DeepEqual(left, right)
}

func compareOrdered(left, right any, op string) (bool, error) {
	// Try numeric comparison.
	lf, lok := toFloat(left)
	rf, rok := toFloat(right)
	if lok && rok {
		switch op {
		case "<":
			return lf < rf, nil
		case ">":
			return lf > rf, nil
		case "<=":
			return lf <= rf, nil
		case ">=":
			return lf >= rf, nil
		}
	}

	// Try string comparison.
	ls, lok := left.(string)
	rs, rok := right.(string)
	if lok && rok {
		switch op {
		case "<":
			return ls < rs, nil
		case ">":
			return ls > rs, nil
		case "<=":
			return ls <= rs, nil
		case ">=":
			return ls >= rs, nil
		}
	}

	return false, fmt.Errorf("cannot compare %T %s %T", left, op, right)
}

func toFloat(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}

func formatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}
