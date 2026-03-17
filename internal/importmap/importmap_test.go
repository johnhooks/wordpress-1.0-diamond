package importmap

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuild(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/prosemirror-model.js": &fstest.MapFile{Data: []byte("export const x = 1;")},
		"vendor/prosemirror-state.js": &fstest.MapFile{Data: []byte("export const y = 2;")},
		"vendor/orderedmap.js":        &fstest.MapFile{Data: []byte("export default {}; ")},
	}

	m, err := Build(fs, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	tests := []struct {
		specifier string
	}{
		{"prosemirror-model"},
		{"prosemirror-state"},
		{"orderedmap"},
	}

	for _, tt := range tests {
		t.Run(tt.specifier, func(t *testing.T) {
			path, ok := m.Imports[tt.specifier]
			if !ok {
				t.Fatalf("missing import for %q", tt.specifier)
			}
			if !strings.HasPrefix(path, "/static/vendor/"+tt.specifier+"-") {
				t.Errorf("expected hashed path, got %q", path)
			}
			if !strings.HasSuffix(path, ".js") {
				t.Errorf("expected .js suffix, got %q", path)
			}
		})
	}
}

func TestBuildHashChanges(t *testing.T) {
	fs1 := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("version 1")},
	}
	fs2 := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("version 2")},
	}

	m1, _ := Build(fs1, "vendor", "/static/vendor/")
	m2, _ := Build(fs2, "vendor", "/static/vendor/")

	if m1.Imports["lib"] == m2.Imports["lib"] {
		t.Error("expected different hashes for different content")
	}
}

func TestBuildHashStable(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("same content")},
	}

	m1, _ := Build(fs, "vendor", "/static/vendor/")
	m2, _ := Build(fs, "vendor", "/static/vendor/")

	if m1.Imports["lib"] != m2.Imports["lib"] {
		t.Error("expected same hash for same content")
	}
}

func TestBuildIgnoresNonJS(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js":    &fstest.MapFile{Data: []byte("export const x = 1;")},
		"vendor/readme.md": &fstest.MapFile{Data: []byte("# readme")},
		"vendor/lib.d.ts":  &fstest.MapFile{Data: []byte("declare const x: number;")},
	}

	m, err := Build(fs, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(m.Imports) != 1 {
		t.Errorf("expected 1 import, got %d: %v", len(m.Imports), m.Imports)
	}
}

func TestBuildHashedMap(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("content")},
	}

	m, _ := Build(fs, "vendor", "/static/vendor/")

	original := "/static/vendor/lib.js"
	hashed, ok := m.Hashed[original]
	if !ok {
		t.Fatalf("missing hashed entry for %q", original)
	}
	if hashed != m.Imports["lib"] {
		t.Errorf("hashed path %q should match import path %q", hashed, m.Imports["lib"])
	}
}

func TestJSON(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/prosemirror-model.js": &fstest.MapFile{Data: []byte("export const x = 1;")},
	}

	m, _ := Build(fs, "vendor", "/static/vendor/")
	j, err := m.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var parsed struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(j), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := parsed.Imports["prosemirror-model"]; !ok {
		t.Error("expected prosemirror-model in JSON imports")
	}
}

func TestTag(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("export const x = 1;")},
	}

	m, _ := Build(fs, "vendor", "/static/vendor/")
	tag, err := m.Tag()
	if err != nil {
		t.Fatalf("Tag: %v", err)
	}

	s := string(tag)
	if !strings.HasPrefix(s, `<script type="importmap">`) {
		t.Errorf("expected script tag prefix, got %q", s[:40])
	}
	if !strings.HasSuffix(s, "</script>") {
		t.Error("expected </script> suffix")
	}
	if !strings.Contains(s, `"imports"`) {
		t.Error("expected imports key in tag")
	}
}

func TestBuildEmptyVendor(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/.gitkeep": &fstest.MapFile{Data: []byte("")},
	}

	m, err := Build(fs, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(m.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(m.Imports))
	}
}
