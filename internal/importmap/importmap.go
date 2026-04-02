package importmap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

const manifestFile = "manifest.json"

// Map holds the computed import map for vendored JavaScript modules.
// Every vendor file is ESM and gets a bare specifier in the import map.
type Map struct {
	// Imports maps bare specifiers to hashed URL paths for the
	// browser's import map resolution.
	Imports map[string]string `json:"imports"`

	// Hashed maps original URL paths to hashed URL paths. The server
	// uses this to resolve incoming requests for hashed URLs back to
	// the original file on disk.
	Hashed map[string]string `json:"hashed"`
}

// Build scans the vendor directory for .js files and builds the
// import map. The bare specifier is the filename without the .js
// extension (e.g., "prosemirror-model.js" → "prosemirror-model").
// Each file is content-hashed for cache busting.
//
// If the vendor directory does not exist or contains no .js files,
// Build returns an empty map.
func Build(staticFS fs.FS, vendorDir, urlPrefix string) (*Map, error) {
	m := &Map{
		Imports: make(map[string]string),
		Hashed:  make(map[string]string),
	}

	entries, err := fs.ReadDir(staticFS, vendorDir)
	if err != nil {
		return m, nil // no vendor directory — empty import map
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}

		filename := entry.Name()
		specifier := strings.TrimSuffix(filename, ".js")

		data, err := fs.ReadFile(staticFS, path.Join(vendorDir, filename))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filename, err)
		}

		hash := sha256.Sum256(data)
		digest := hex.EncodeToString(hash[:])
		hashedPath := urlPrefix + specifier + "-" + digest + ".js"

		m.Imports[specifier] = hashedPath
		m.Hashed[urlPrefix+filename] = hashedPath
	}

	return m, nil
}

// WriteManifest writes the computed map to a JSON file. The file is
// human-readable and can be committed to source control.
func (m *Map) WriteManifest(filePath string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filePath, data, 0644)
}

// LoadManifest reads a pre-built manifest from disk.
func LoadManifest(filePath string) (*Map, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var m Map
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest %s: %w", filePath, err)
	}
	if m.Imports == nil {
		m.Imports = make(map[string]string)
	}
	if m.Hashed == nil {
		m.Hashed = make(map[string]string)
	}
	return &m, nil
}

// Load tries to read a pre-built manifest. If the manifest file does
// not exist, it falls back to building from vendor files on disk.
func Load(publicDir, vendorDir, urlPrefix string) (*Map, error) {
	manifestPath := path.Join(publicDir, vendorDir, manifestFile)
	m, err := LoadManifest(manifestPath)
	if err == nil {
		return m, nil
	}
	return Build(os.DirFS(publicDir), vendorDir, urlPrefix)
}

// HeadHTML returns the <script type="importmap"> tag for the document
// head. Returns empty string if there are no vendor assets.
func (m *Map) HeadHTML() template.HTML {
	if m == nil || len(m.Imports) == 0 {
		return ""
	}

	keys := make([]string, 0, len(m.Imports))
	for k := range m.Imports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(`<script type="importmap">{"imports":{`)
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(m.Imports[k])
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
	}
	b.WriteString(`}}</script>`)

	return template.HTML(b.String())
}
