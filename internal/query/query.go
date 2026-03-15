package query

import (
	"fmt"
	"math"
	"strings"
)

// Op represents a filter operator.
type Op string

const (
	Is     Op = "is"
	IsNot  Op = "isNot"
	IsAny  Op = "isAny"
	IsNone Op = "isNone"
)

// Filter represents a single filter condition.
type Filter struct {
	Field    string
	Operator Op
	Value    any
}

// Sort represents a sort directive.
type Sort struct {
	Field     string
	Direction string // "asc" or "desc"
}

// Query represents a paginated, filterable, searchable query.
type Query struct {
	Filters []Filter
	Sort    *Sort
	Page    int
	PerPage int
	Search  string
}

// GetPage returns the page number, defaulting to 1.
func (q Query) GetPage() int {
	if q.Page < 1 {
		return 1
	}
	return q.Page
}

// GetPerPage returns the per-page limit, defaulting to 20.
func (q Query) GetPerPage() int {
	if q.PerPage < 1 {
		return 20
	}
	return q.PerPage
}

// Offset returns the SQL OFFSET for the current page.
func (q Query) Offset() int {
	return (q.GetPage() - 1) * q.GetPerPage()
}

// Result is a paginated result set.
type Result[T any] struct {
	Items      []T
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

// NewResult creates a Result with computed TotalPages.
func NewResult[T any](items []T, total, page, perPage int) *Result[T] {
	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}
	return &Result[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}

// ApplyFilters converts filters into WHERE clauses using a field mapping.
// Unknown fields are silently skipped.
func ApplyFilters(filters []Filter, fieldMap map[string]string) (whereClauses []string, args []any) {
	for _, f := range filters {
		col, ok := fieldMap[f.Field]
		if !ok {
			continue
		}
		switch f.Operator {
		case Is:
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
			args = append(args, f.Value)
		case IsNot:
			whereClauses = append(whereClauses, fmt.Sprintf("%s != ?", col))
			args = append(args, f.Value)
		case IsAny:
			vals, ok := toSlice(f.Value)
			if !ok || len(vals) == 0 {
				continue
			}
			placeholders := make([]string, len(vals))
			for i, v := range vals {
				placeholders[i] = "?"
				args = append(args, v)
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ",")))
		case IsNone:
			vals, ok := toSlice(f.Value)
			if !ok || len(vals) == 0 {
				continue
			}
			placeholders := make([]string, len(vals))
			for i, v := range vals {
				placeholders[i] = "?"
				args = append(args, v)
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s NOT IN (%s)", col, strings.Join(placeholders, ",")))
		}
	}
	return
}

// ApplySort returns an ORDER BY clause. Falls back to defaultSort if sort is
// nil or the field is not in the mapping.
func ApplySort(sort *Sort, fieldMap map[string]string, defaultSort string) string {
	if sort == nil {
		return defaultSort
	}
	col, ok := fieldMap[sort.Field]
	if !ok {
		return defaultSort
	}
	dir := "ASC"
	if strings.EqualFold(sort.Direction, "desc") {
		dir = "DESC"
	}
	return fmt.Sprintf("%s %s", col, dir)
}

// ApplySearch returns a WHERE clause fragment that matches search across
// multiple columns with LIKE. Returns empty string and nil args if search is empty.
func ApplySearch(search string, columns []string) (clause string, args []any) {
	if search == "" || len(columns) == 0 {
		return "", nil
	}
	like := "%" + search + "%"
	parts := make([]string, len(columns))
	for i, col := range columns {
		parts[i] = fmt.Sprintf("%s LIKE ?", col)
		args = append(args, like)
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

// toSlice attempts to convert a value to []any.
func toSlice(v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return s, true
	case []int64:
		out := make([]any, len(s))
		for i, val := range s {
			out[i] = val
		}
		return out, true
	case []string:
		out := make([]any, len(s))
		for i, val := range s {
			out[i] = val
		}
		return out, true
	case []int:
		out := make([]any, len(s))
		for i, val := range s {
			out[i] = val
		}
		return out, true
	default:
		return nil, false
	}
}
