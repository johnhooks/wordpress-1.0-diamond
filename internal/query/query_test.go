package query

import (
	"testing"
)

func TestGetPage(t *testing.T) {
	tests := []struct {
		page int
		want int
	}{
		{0, 1},
		{-1, 1},
		{1, 1},
		{3, 3},
	}
	for _, tt := range tests {
		q := Query{Page: tt.page}
		if got := q.GetPage(); got != tt.want {
			t.Errorf("GetPage(%d) = %d, want %d", tt.page, got, tt.want)
		}
	}
}

func TestGetPerPage(t *testing.T) {
	tests := []struct {
		perPage int
		want    int
	}{
		{0, 20},
		{-1, 20},
		{10, 10},
		{50, 50},
	}
	for _, tt := range tests {
		q := Query{PerPage: tt.perPage}
		if got := q.GetPerPage(); got != tt.want {
			t.Errorf("GetPerPage(%d) = %d, want %d", tt.perPage, got, tt.want)
		}
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		page, perPage, want int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 5, 10},
		{0, 10, 0}, // defaults to page 1
	}
	for _, tt := range tests {
		q := Query{Page: tt.page, PerPage: tt.perPage}
		if got := q.Offset(); got != tt.want {
			t.Errorf("Offset(page=%d, perPage=%d) = %d, want %d", tt.page, tt.perPage, got, tt.want)
		}
	}
}

func TestNewResult(t *testing.T) {
	r := NewResult([]string{"a", "b"}, 15, 2, 5)
	if r.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", r.TotalPages)
	}
	if r.Total != 15 {
		t.Errorf("Total = %d, want 15", r.Total)
	}
	if len(r.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(r.Items))
	}

	// Edge: 0 perPage
	r2 := NewResult([]string{}, 0, 1, 0)
	if r2.TotalPages != 0 {
		t.Errorf("TotalPages with 0 perPage = %d, want 0", r2.TotalPages)
	}

	// Edge: total not evenly divisible
	r3 := NewResult([]string{}, 11, 1, 5)
	if r3.TotalPages != 3 {
		t.Errorf("TotalPages ceil(11/5) = %d, want 3", r3.TotalPages)
	}
}

func TestApplyFilters_Is(t *testing.T) {
	fieldMap := map[string]string{
		"status": "p.post_status",
		"type":   "p.post_type",
	}

	filters := []Filter{
		{Field: "status", Operator: Is, Value: "publish"},
		{Field: "type", Operator: Is, Value: "post"},
	}

	clauses, args := ApplyFilters(filters, fieldMap)
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
	if clauses[0] != "p.post_status = ?" {
		t.Errorf("clause[0] = %q", clauses[0])
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "publish" {
		t.Errorf("args[0] = %v", args[0])
	}
}

func TestApplyFilters_IsNot(t *testing.T) {
	fieldMap := map[string]string{"status": "post_status"}
	clauses, args := ApplyFilters([]Filter{
		{Field: "status", Operator: IsNot, Value: "trash"},
	}, fieldMap)
	if len(clauses) != 1 || clauses[0] != "post_status != ?" {
		t.Errorf("clause = %v", clauses)
	}
	if len(args) != 1 || args[0] != "trash" {
		t.Errorf("args = %v", args)
	}
}

func TestApplyFilters_IsAny(t *testing.T) {
	fieldMap := map[string]string{"status": "post_status"}
	clauses, args := ApplyFilters([]Filter{
		{Field: "status", Operator: IsAny, Value: []string{"publish", "draft"}},
	}, fieldMap)
	if len(clauses) != 1 || clauses[0] != "post_status IN (?,?)" {
		t.Errorf("clause = %v", clauses)
	}
	if len(args) != 2 {
		t.Errorf("args = %v", args)
	}
}

func TestApplyFilters_IsNone(t *testing.T) {
	fieldMap := map[string]string{"id": "ID"}
	clauses, args := ApplyFilters([]Filter{
		{Field: "id", Operator: IsNone, Value: []int64{1, 2, 3}},
	}, fieldMap)
	if len(clauses) != 1 || clauses[0] != "ID NOT IN (?,?,?)" {
		t.Errorf("clause = %v", clauses)
	}
	if len(args) != 3 {
		t.Errorf("args = %v", args)
	}
}

func TestApplyFilters_UnknownFieldSkipped(t *testing.T) {
	fieldMap := map[string]string{"status": "post_status"}
	clauses, args := ApplyFilters([]Filter{
		{Field: "unknown_field", Operator: Is, Value: "x"},
	}, fieldMap)
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses for unknown field, got %d", len(clauses))
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

func TestApplyFilters_EmptyFilters(t *testing.T) {
	clauses, args := ApplyFilters(nil, map[string]string{"x": "y"})
	if len(clauses) != 0 || len(args) != 0 {
		t.Errorf("expected empty results for nil filters")
	}
}

func TestApplySort(t *testing.T) {
	fieldMap := map[string]string{
		"title": "post_title",
		"date":  "post_date",
	}

	// Mapped field
	got := ApplySort(&Sort{Field: "title", Direction: "asc"}, fieldMap, "post_date DESC")
	if got != "post_title ASC" {
		t.Errorf("ApplySort = %q, want 'post_title ASC'", got)
	}

	// Desc
	got = ApplySort(&Sort{Field: "date", Direction: "DESC"}, fieldMap, "post_date DESC")
	if got != "post_date DESC" {
		t.Errorf("ApplySort desc = %q", got)
	}

	// Nil sort → default
	got = ApplySort(nil, fieldMap, "post_date DESC")
	if got != "post_date DESC" {
		t.Errorf("ApplySort nil = %q", got)
	}

	// Unknown field → default
	got = ApplySort(&Sort{Field: "unknown"}, fieldMap, "post_date DESC")
	if got != "post_date DESC" {
		t.Errorf("ApplySort unknown = %q", got)
	}
}

func TestApplySearch(t *testing.T) {
	clause, args := ApplySearch("hello", []string{"title", "content"})
	if clause != "(title LIKE ? OR content LIKE ?)" {
		t.Errorf("clause = %q", clause)
	}
	if len(args) != 2 || args[0] != "%hello%" {
		t.Errorf("args = %v", args)
	}

	// Empty search
	clause, args = ApplySearch("", []string{"title"})
	if clause != "" || args != nil {
		t.Errorf("expected empty for empty search")
	}

	// No columns
	clause, args = ApplySearch("hello", nil)
	if clause != "" || args != nil {
		t.Errorf("expected empty for no columns")
	}
}
