package importmap

import (
	"crypto/sha512"
	"encoding/base64"
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

	m, err := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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
		"vendor/lib.js":    &fstest.MapFile{Data: []byte("export default {};")},
		"vendor/deps.json": &fstest.MapFile{Data: []byte(`{}`)},
		"vendor/readme.md": &fstest.MapFile{Data: []byte("# readme")},
		"vendor/.gitkeep":  &fstest.MapFile{Data: []byte("")},
	}

	m, err := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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

	m1, _ := Build(fs1, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
	m2, _ := Build(fs2, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})

	if m1.Imports["lib"] == m2.Imports["lib"] {
		t.Error("expected different hashes for different content")
	}
}

func TestBuildHashStable(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("same content")},
	}

	m1, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
	m2, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})

	if m1.Imports["lib"] != m2.Imports["lib"] {
		t.Error("expected same hash for same content")
	}
}

func TestBuildHashedMap(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("content")},
	}

	m, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})

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

	m, err := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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

	m, err := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(m.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(m.Imports))
	}
}

func TestBuildIntegrity(t *testing.T) {
	content := []byte("export const x = 1;")
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: content},
	}

	m, err := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	hashedPath := m.Imports["lib"]
	sri, ok := m.Integrity[hashedPath]
	if !ok {
		t.Fatalf("missing integrity for hashed path %q", hashedPath)
	}
	if !strings.HasPrefix(sri, "sha384-") {
		t.Errorf("expected sha384- prefix, got %q", sri)
	}

	// Verify the SRI value is correct.
	hash := sha512.Sum384(content)
	expected := "sha384-" + base64.StdEncoding.EncodeToString(hash[:])
	if sri != expected {
		t.Errorf("integrity mismatch\n  got  %s\n  want %s", sri, expected)
	}
}

func TestBuildMultipleDirs(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js":    &fstest.MapFile{Data: []byte("vendor lib")},
		"js/the-editor.js": &fstest.MapFile{Data: []byte("local editor")},
	}

	m, err := Build(fs,
		AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"},
		AssetDir{Dir: "js", URLPrefix: "/static/js/"},
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(m.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d: %v", len(m.Imports), m.Imports)
	}
	if _, ok := m.Imports["lib"]; !ok {
		t.Error("missing vendor import 'lib'")
	}
	if _, ok := m.Imports["the-editor"]; !ok {
		t.Error("missing local import 'the-editor'")
	}

	// Both should have integrity entries.
	for spec, path := range m.Imports {
		if _, ok := m.Integrity[path]; !ok {
			t.Errorf("missing integrity for %q (%s)", spec, path)
		}
	}
}

func TestBuildMultipleDirsMissingOne(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("vendor lib")},
	}

	m, err := Build(fs,
		AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"},
		AssetDir{Dir: "js", URLPrefix: "/static/js/"},
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(m.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(m.Imports))
	}
}

func TestManifestRoundTrip(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js":      &fstest.MapFile{Data: []byte("export default {};")},
		"vendor/htmx.org.js": &fstest.MapFile{Data: []byte("var htmx = {};")},
	}

	original, err := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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
	if len(loaded.Integrity) != len(original.Integrity) {
		t.Errorf("integrity: got %d, want %d", len(loaded.Integrity), len(original.Integrity))
	}
	for k, v := range original.Integrity {
		if loaded.Integrity[k] != v {
			t.Errorf("integrity[%q]: got %q, want %q", k, loaded.Integrity[k], v)
		}
	}
}

func TestManifestIsReadableJSON(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("content")},
	}

	m, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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

	m, err := Load(dir,
		AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"},
	)
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
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(vendorDir, "lib.js"), []byte("content"), 0644)

	m, err := Load(dir,
		AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"},
	)
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

	m, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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

func TestHeadHTMLIntegrity(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/alpha.js": &fstest.MapFile{Data: []byte("a")},
		"vendor/beta.js":  &fstest.MapFile{Data: []byte("b")},
	}

	m, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
	h := string(m.HeadHTML())

	if !strings.Contains(h, `"integrity":{`) {
		t.Error("expected integrity block in import map")
	}
	if !strings.Contains(h, `"sha384-`) {
		t.Error("expected sha384- values in integrity block")
	}

	// Integrity keys should be sorted within the integrity section.
	intStart := strings.Index(h, `"integrity"`)
	intAlpha := strings.Index(h[intStart:], m.Imports["alpha"])
	intBeta := strings.Index(h[intStart:], m.Imports["beta"])
	if intAlpha > intBeta {
		t.Error("integrity entries should be sorted")
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

func TestHeadHTMLNoIntegrityWhenEmpty(t *testing.T) {
	m := &Map{
		Imports:   map[string]string{"lib": "/static/vendor/lib-abc.js"},
		Hashed:    map[string]string{},
		Integrity: map[string]string{},
	}
	h := string(m.HeadHTML())

	if strings.Contains(h, "integrity") {
		t.Error("expected no integrity block when Integrity map is empty")
	}
}

func TestHeadHTMLDeterministic(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/alpha.js": &fstest.MapFile{Data: []byte("a")},
		"vendor/beta.js":  &fstest.MapFile{Data: []byte("b")},
		"vendor/gamma.js": &fstest.MapFile{Data: []byte("g")},
	}

	m, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
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

func TestBuildDev(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js":    &fstest.MapFile{Data: []byte("vendor lib")},
		"js/the-editor.js": &fstest.MapFile{Data: []byte("editor")},
	}

	m, err := BuildDev(
		DevSource{FS: fs, ScanDir: "vendor", URLPrefix: "/static/vendor/"},
		DevSource{FS: fs, ScanDir: "js", URLPrefix: "/static/js/"},
	)
	if err != nil {
		t.Fatalf("BuildDev: %v", err)
	}

	// Bare specifiers map to unhashed URLs.
	if m.Imports["lib"] != "/static/vendor/lib.js" {
		t.Errorf("lib: got %q, want /static/vendor/lib.js", m.Imports["lib"])
	}
	if m.Imports["the-editor"] != "/static/js/the-editor.js" {
		t.Errorf("the-editor: got %q, want /static/js/the-editor.js", m.Imports["the-editor"])
	}

	// No hashing or integrity in dev.
	if len(m.Hashed) != 0 {
		t.Errorf("expected empty Hashed, got %d entries", len(m.Hashed))
	}
	if len(m.Integrity) != 0 {
		t.Errorf("expected empty Integrity, got %d entries", len(m.Integrity))
	}
}

func TestBuildDevMissingDir(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("vendor lib")},
	}

	m, err := BuildDev(
		DevSource{FS: fs, ScanDir: "vendor", URLPrefix: "/static/vendor/"},
		DevSource{FS: fs, ScanDir: "js", URLPrefix: "/static/js/"},
	)
	if err != nil {
		t.Fatalf("BuildDev: %v", err)
	}

	if len(m.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(m.Imports))
	}
}

func TestBuildDevHeadHTML(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("content")},
	}

	m, _ := BuildDev(DevSource{FS: fs, ScanDir: "vendor", URLPrefix: "/static/vendor/"})
	h := string(m.HeadHTML())

	// Should have imports but no integrity block.
	if !strings.Contains(h, `"lib":"/static/vendor/lib.js"`) {
		t.Errorf("expected unhashed URL in import map, got %s", h)
	}
	if strings.Contains(h, "integrity") {
		t.Error("expected no integrity block in dev mode")
	}
}

func TestBuildSHA384HashLength(t *testing.T) {
	fs := fstest.MapFS{
		"vendor/lib.js": &fstest.MapFile{Data: []byte("content")},
	}

	m, _ := Build(fs, AssetDir{Dir: "vendor", URLPrefix: "/static/vendor/"})
	path := m.Imports["lib"]

	// SHA-384 hex digest is 96 characters. Path format:
	// /static/vendor/lib-<96 hex chars>.js
	prefix := "/static/vendor/lib-"
	suffix := ".js"
	hash := strings.TrimPrefix(path, prefix)
	hash = strings.TrimSuffix(hash, suffix)
	if len(hash) != 96 {
		t.Errorf("expected 96-char SHA-384 hex digest, got %d chars: %s", len(hash), hash)
	}
}
