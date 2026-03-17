package permalink

import (
	"testing"
	"time"
)

var testDate = time.Date(2004, 1, 3, 14, 30, 0, 0, time.UTC)

func TestPost_PrettyPermalinks(t *testing.T) {
	l := &Linker{
		SiteURL:   "http://localhost:3000",
		Structure: "/%year%/%monthnum%/%day%/%postname%/",
	}

	got := l.Post(42, testDate, "hello-world")
	want := "http://localhost:3000/2004/01/03/hello-world/"
	if got != want {
		t.Errorf("Post() = %q, want %q", got, want)
	}
}

func TestPost_PostIDStructure(t *testing.T) {
	l := &Linker{
		SiteURL:   "http://localhost:3000",
		Structure: "/archives/%post_id%/",
	}

	got := l.Post(42, testDate, "hello-world")
	want := "http://localhost:3000/archives/42/"
	if got != want {
		t.Errorf("Post() = %q, want %q", got, want)
	}
}

func TestPost_QueryStringFallback(t *testing.T) {
	l := &Linker{
		SiteURL:   "http://localhost:3000",
		Structure: "",
	}

	got := l.Post(42, testDate, "hello-world")
	want := "http://localhost:3000/?p=42"
	if got != want {
		t.Errorf("Post() = %q, want %q", got, want)
	}
}

func TestPostPath(t *testing.T) {
	tests := []struct {
		name      string
		structure string
		want      string
	}{
		{"pretty", "/%year%/%monthnum%/%day%/%postname%/", "/2004/01/03/hello-world/"},
		{"empty", "", "/?p=42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Linker{SiteURL: "http://localhost:3000", Structure: tt.structure}
			got := l.PostPath(42, testDate, "hello-world")
			if got != tt.want {
				t.Errorf("PostPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMonth(t *testing.T) {
	tests := []struct {
		name      string
		structure string
		want      string
	}{
		{"pretty", "/%year%/%monthnum%/%day%/%postname%/", "http://localhost:3000/2004/01/"},
		{"empty", "", "http://localhost:3000/?m=200401"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Linker{SiteURL: "http://localhost:3000", Structure: tt.structure}
			got := l.Month(2004, 1)
			if got != tt.want {
				t.Errorf("Month() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDay(t *testing.T) {
	tests := []struct {
		name      string
		structure string
		want      string
	}{
		{"pretty", "/%year%/%monthnum%/%day%/%postname%/", "http://localhost:3000/2004/01/03/"},
		{"empty", "", "http://localhost:3000/?m=20040103"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Linker{SiteURL: "http://localhost:3000", Structure: tt.structure}
			got := l.Day(2004, 1, 3)
			if got != tt.want {
				t.Errorf("Day() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCategory(t *testing.T) {
	tests := []struct {
		name      string
		structure string
		want      string
	}{
		{"pretty", "/%year%/%monthnum%/%day%/%postname%/", "http://localhost:3000/category/tech/"},
		{"with front", "/blog/%postname%/", "http://localhost:3000/blog/category/tech/"},
		{"empty", "", "http://localhost:3000/?cat=5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Linker{SiteURL: "http://localhost:3000", Structure: tt.structure}
			got := l.Category(5, "tech")
			if got != tt.want {
				t.Errorf("Category() = %q, want %q", got, tt.want)
			}
		})
	}
}
