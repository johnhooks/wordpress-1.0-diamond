package permalink

import (
	"testing"
	"time"
)

func TestRouter_Resolve(t *testing.T) {
	tests := []struct {
		name      string
		structure string
		path      string
		wantType  RouteType
		wantNil   bool
		check     func(t *testing.T, m *Match)
	}{
		// Posts
		{
			name:      "classic post",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/01/03/hello-world/",
			wantType:  RoutePost,
			check: func(t *testing.T, m *Match) {
				if m.PostName != "hello-world" {
					t.Errorf("PostName = %q", m.PostName)
				}
			},
		},
		{
			name:      "postname-only post",
			structure: "/%postname%/",
			path:      "/hello-world/",
			wantType:  RoutePost,
		},
		{
			name:      "post_id post",
			structure: "/archives/%post_id%/",
			path:      "/archives/42/",
			wantType:  RoutePost,
			check: func(t *testing.T, m *Match) {
				if m.PostID != "42" {
					t.Errorf("PostID = %q", m.PostID)
				}
			},
		},
		// Categories
		{
			name:      "category with date structure",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/category/tech/",
			wantType:  RouteCategory,
			check: func(t *testing.T, m *Match) {
				if m.Category != "tech" {
					t.Errorf("Category = %q", m.Category)
				}
			},
		},
		{
			name:      "category with front",
			structure: "/blog/%postname%/",
			path:      "/blog/category/tech/",
			wantType:  RouteCategory,
		},
		// Date archives
		{
			name:      "day archive",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/01/03/",
			wantType:  RouteDayArchive,
			check: func(t *testing.T, m *Match) {
				if m.Year != "2004" || m.Month != "01" || m.Day != "03" {
					t.Errorf("date = %s/%s/%s", m.Year, m.Month, m.Day)
				}
			},
		},
		{
			name:      "month archive",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/01/",
			wantType:  RouteMonthArchive,
		},
		{
			name:      "year archive",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/",
			wantType:  RouteYearArchive,
		},
		// With postname-only, any single-segment path matches as a post.
		// The handler will 404 if no post with that slug exists.
		{
			name:      "postname-only matches any segment as post",
			structure: "/%postname%/",
			path:      "/2004/",
			wantType:  RoutePost,
			check: func(t *testing.T, m *Match) {
				if m.PostName != "2004" {
					t.Errorf("PostName = %q", m.PostName)
				}
			},
		},
		// Non-matching
		{
			name:      "no match",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/some/random/path/",
			wantNil:   true,
		},
		{
			name:      "empty structure — nothing resolves",
			structure: "",
			path:      "/2004/01/03/hello-world/",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.structure)
			route := router.Resolve(tt.path)

			if tt.wantNil {
				if route != nil {
					t.Errorf("Resolve(%q) = %+v, want nil", tt.path, route)
				}
				return
			}

			if route == nil {
				t.Fatalf("Resolve(%q) = nil", tt.path)
			}
			if route.Type != tt.wantType {
				t.Errorf("Type = %d, want %d", route.Type, tt.wantType)
			}
			if tt.check != nil {
				tt.check(t, route.Match)
			}
		})
	}
}

func TestRouter_RoundTrip(t *testing.T) {
	structure := "/%year%/%monthnum%/%day%/%postname%/"
	linker := &Linker{SiteURL: "http://localhost:3000", Structure: structure}
	router := NewRouter(structure)
	date := time.Date(2004, 1, 3, 0, 0, 0, 0, time.UTC)

	// Post round-trip
	path := linker.PostPath(42, date, "hello-world")
	route := router.Resolve(path)
	if route == nil || route.Type != RoutePost {
		t.Fatalf("post path %q did not resolve as RoutePost", path)
	}
	if route.Match.PostName != "hello-world" {
		t.Errorf("PostName = %q", route.Match.PostName)
	}

	// Category round-trip
	catURL := linker.Category(5, "tech")
	// Strip the site URL to get the path
	catPath := catURL[len("http://localhost:3000"):]
	route = router.Resolve(catPath)
	if route == nil || route.Type != RouteCategory {
		t.Fatalf("category path %q did not resolve as RouteCategory", catPath)
	}
	if route.Match.Category != "tech" {
		t.Errorf("Category = %q", route.Match.Category)
	}

	// Month archive round-trip
	monthURL := linker.Month(2004, 1)
	monthPath := monthURL[len("http://localhost:3000"):]
	route = router.Resolve(monthPath)
	if route == nil || route.Type != RouteMonthArchive {
		t.Fatalf("month path %q did not resolve as RouteMonthArchive", monthPath)
	}

	// Day archive round-trip
	dayURL := linker.Day(2004, 1, 3)
	dayPath := dayURL[len("http://localhost:3000"):]
	route = router.Resolve(dayPath)
	if route == nil || route.Type != RouteDayArchive {
		t.Fatalf("day path %q did not resolve as RouteDayArchive", dayPath)
	}
}
