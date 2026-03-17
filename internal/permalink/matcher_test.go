package permalink

import (
	"testing"
	"time"
)

func TestMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		structure string
		path      string
		want      *Match
	}{
		{
			name:      "classic year/month/day/postname",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/01/03/hello-world/",
			want:      &Match{Year: "2004", Month: "01", Day: "03", PostName: "hello-world"},
		},
		{
			name:      "postname only",
			structure: "/%postname%/",
			path:      "/hello-world/",
			want:      &Match{PostName: "hello-world"},
		},
		{
			name:      "archives with post_id",
			structure: "/archives/%post_id%/",
			path:      "/archives/42/",
			want:      &Match{PostID: "42"},
		},
		{
			name:      "month and name",
			structure: "/%year%/%monthnum%/%postname%/",
			path:      "/2004/01/hello-world/",
			want:      &Match{Year: "2004", Month: "01", PostName: "hello-world"},
		},
		{
			name:      "custom literal prefix",
			structure: "/blog/%year%/%postname%/",
			path:      "/blog/2004/hello-world/",
			want:      &Match{Year: "2004", PostName: "hello-world"},
		},
		{
			name:      "single-digit month and day",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/1/3/hello-world/",
			want:      &Match{Year: "2004", Month: "1", Day: "3", PostName: "hello-world"},
		},
		{
			name:      "no match — wrong segment count",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/2004/01/hello-world/",
			want:      nil,
		},
		{
			name:      "no match — letters where digits expected",
			structure: "/%year%/%monthnum%/%day%/%postname%/",
			path:      "/abcd/01/03/hello-world/",
			want:      nil,
		},
		{
			name:      "no match — wrong literal prefix",
			structure: "/archives/%post_id%/",
			path:      "/blog/42/",
			want:      nil,
		},
		{
			name:      "empty structure — never matches",
			structure: "",
			path:      "/2004/01/03/hello-world/",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMatcher(tt.structure)
			got := m.match(tt.path)

			if tt.want == nil {
				if got != nil {
					t.Errorf("Match(%q) = %+v, want nil", tt.path, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("Match(%q) = nil, want %+v", tt.path, tt.want)
			}

			if got.Year != tt.want.Year {
				t.Errorf("Year = %q, want %q", got.Year, tt.want.Year)
			}
			if got.Month != tt.want.Month {
				t.Errorf("Month = %q, want %q", got.Month, tt.want.Month)
			}
			if got.Day != tt.want.Day {
				t.Errorf("Day = %q, want %q", got.Day, tt.want.Day)
			}
			if got.PostName != tt.want.PostName {
				t.Errorf("PostName = %q, want %q", got.PostName, tt.want.PostName)
			}
			if got.PostID != tt.want.PostID {
				t.Errorf("PostID = %q, want %q", got.PostID, tt.want.PostID)
			}
		})
	}
}

func TestMatch_ValidateDate(t *testing.T) {
	date := time.Date(2004, 1, 3, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name  string
		match Match
		want  bool
	}{
		{
			name:  "all date fields correct",
			match: Match{Year: "2004", Month: "01", Day: "03"},
			want:  true,
		},
		{
			name:  "single-digit month and day correct",
			match: Match{Year: "2004", Month: "1", Day: "3"},
			want:  true,
		},
		{
			name:  "wrong year",
			match: Match{Year: "2005", Month: "01", Day: "03"},
			want:  false,
		},
		{
			name:  "wrong month",
			match: Match{Year: "2004", Month: "02", Day: "03"},
			want:  false,
		},
		{
			name:  "wrong day",
			match: Match{Year: "2004", Month: "01", Day: "04"},
			want:  false,
		},
		{
			name:  "no date fields — always valid",
			match: Match{PostName: "hello-world"},
			want:  true,
		},
		{
			name:  "time fields correct",
			match: Match{Hour: "14", Minute: "30", Second: "45"},
			want:  true,
		},
		{
			name:  "wrong hour",
			match: Match{Hour: "15"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.match.ValidateDate(date)
			if got != tt.want {
				t.Errorf("ValidateDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	date := time.Date(2004, 1, 3, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		structure string
		id        int64
		slug      string
	}{
		{"classic", "/%year%/%monthnum%/%day%/%postname%/", 42, "hello-world"},
		{"postname only", "/%postname%/", 42, "hello-world"},
		{"archives by id", "/archives/%post_id%/", 42, "hello-world"},
		{"month and name", "/%year%/%monthnum%/%postname%/", 7, "my-post"},
		{"blog prefix", "/blog/%year%/%postname%/", 99, "some-slug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linker := &Linker{SiteURL: "http://localhost:3000", Structure: tt.structure}
			matcher := newMatcher(tt.structure)

			path := linker.PostPath(tt.id, date, tt.slug)
			m := matcher.match(path)
			if m == nil {
				t.Fatalf("match(%q) = nil, want match", path)
			}

			if m.PostName != "" && m.PostName != tt.slug {
				t.Errorf("PostName = %q, want %q", m.PostName, tt.slug)
			}
			if m.PostID != "" && m.PostID != "42" {
				t.Errorf("PostID = %q, want %q", m.PostID, "42")
			}
			if !m.ValidateDate(date) {
				t.Error("ValidateDate() = false, want true")
			}
		})
	}
}
