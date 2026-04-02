package importmap

import (
	"os"
	"path/filepath"
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

	if len(m.Imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(m.Imports))
	}

	for _, specifier := range []string{"prosemirror-model", "prosemirror-state", "orderedmap"} {
		t.Run(specifier, func(t *testing.T) {
			path, ok := m.Imports[specifier]
			if !ok {
				t.Fatalf("missing import for %q", specifier)
			}
			if !strings.HasPrefix(path, "/static/vendor/"+specifier+"-") {
				t.Errorf("expected hashed path, got %q", path)
			}
			if !strings.HasSuffix(path, ".js") {
				t.Errorf("expected .js suffix, got %q", path)
			}
		})
	}
}

func TestBuildIgnoresNonJS(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js":       &fstest.MapFile{Data: []byte("export default {};")},
		"vendor/pins.json":    &fstest.MapFile{Data: []byte(`{}`)},
		"vendor/readme.md":    &fstest.MapFile{Data: []byte("# readme")},
		"vendor/.gitkeep":     &fstest.MapFile{Data: []byte("")},
	}

	m, err := Build(fs, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(m.Imports) != 1 {
		t.Errorf("expected 1 import, got %d: %v", len(m.Imports), m.Imports)
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

func TestBuildNoVendorDir(t *testing.T) {
	fs := fstest.MapFS{}

	m, err := Build(fs, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(m.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(m.Imports))
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

func TestManifestRoundTrip(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js":      &fstest.MapFile{Data: []byte("export default {};")},
		"vendor/htmx.org.js": &fstest.MapFile{Data: []byte("var htmx = {};")},
	}

	original, err := Build(fs, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := original.WriteManifest(manifestPath); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if len(loaded.Imports) != len(original.Imports) {
		t.Errorf("imports: got %d, want %d", len(loaded.Imports), len(original.Imports))
	}
	for k, v := range original.Imports {
		if loaded.Imports[k] != v {
			t.Errorf("imports[%q]: got %q, want %q", k, loaded.Imports[k], v)
		}
	}
	if len(loaded.Hashed) != len(original.Hashed) {
		t.Errorf("hashed: got %d, want %d", len(loaded.Hashed), len(original.Hashed))
	}
}

func TestManifestIsReadableJSON(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("content")},
	}

	m, _ := Build(fs, "vendor", "/static/vendor/")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := m.WriteManifest(manifestPath); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	data, _ := os.ReadFile(manifestPath)
	s := string(data)

	if !strings.Contains(s, "\n  ") {
		t.Error("expected indented JSON")
	}
	if s[len(s)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestLoadFallsBackToBuild(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(vendorDir, "lib.js"), []byte("content"), 0644)

	m, err := Load(dir, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := m.Imports["lib"]; !ok {
		t.Error("expected lib in imports after fallback build")
	}
}

func TestLoadPrefersManifest(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifest := `{"imports":{"lib":"/static/vendor/lib-frommanifest.js"},"hashed":{"/static/vendor/lib.js":"/static/vendor/lib-frommanifest.js"}}`
	os.WriteFile(filepath.Join(vendorDir, "manifest.json"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(vendorDir, "lib.js"), []byte("content"), 0644)

	m, err := Load(dir, "vendor", "/static/vendor/")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Imports["lib"] != "/static/vendor/lib-frommanifest.js" {
		t.Errorf("expected manifest value, got %q", m.Imports["lib"])
	}
}

func TestHeadHTML(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("export const x = 1;")},
	}

	m, _ := Build(fs, "vendor", "/static/vendor/")
	h := string(m.HeadHTML())

	if !strings.HasPrefix(h, `<script type="importmap">`) {
		t.Errorf("expected importmap script tag, got %q", h)
	}
	if !strings.HasSuffix(h, "</script>") {
		t.Error("expected </script> suffix")
	}
	if !strings.Contains(h, `"lib"`) {
		t.Error("expected lib specifier in import map")
	}
}

func TestHeadHTMLEmpty(t *testing.T) {
	m := &Map{Imports: make(map[string]string)}
	h := m.HeadHTML()
	if h != "" {
		t.Errorf("expected empty head HTML for empty map, got %q", h)
	}
}

func TestHeadHTMLNil(t *testing.T) {
	var m *Map
	if m.HeadHTML() != "" {
		t.Error("expected empty head HTML for nil map")
	}
}

func TestHeadHTMLDeterministic(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/alpha.js": &fstest.MapFile{Data: []byte("a")},
		"vendor/beta.js":  &fstest.MapFile{Data: []byte("b")},
		"vendor/gamma.js": &fstest.MapFile{Data: []byte("g")},
	}

	m, _ := Build(fs, "vendor", "/static/vendor/")
	h1 := string(m.HeadHTML())
	h2 := string(m.HeadHTML())

	if h1 != h2 {
		t.Error("HeadHTML should be deterministic")
	}
	alphaIdx := strings.Index(h1, "alpha")
	betaIdx := strings.Index(h1, "beta")
	gammaIdx := strings.Index(h1, "gamma")
	if alphaIdx > betaIdx || betaIdx > gammaIdx {
		t.Error("import map entries should be in alphabetical order")
	}
}
