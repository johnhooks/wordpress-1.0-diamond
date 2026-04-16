package importmap

import (
	"crypto/sha512"
	"encoding/base64"
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

// AssetDir pairs a directory name within the static filesystem with
// the URL prefix used to serve its files.
type AssetDir struct {
	Dir       string
	URLPrefix string
}

// Map holds the computed import map for JavaScript modules.
// Every asset file is ESM and gets a bare specifier in the import map.
type Map struct {
	// Imports maps bare specifiers to hashed URL paths for the
	// browser's import map resolution.
	Imports map[string]string `json:"imports"`

	// Hashed maps original URL paths to hashed URL paths. The server
	// uses this to resolve incoming requests for hashed URLs back to
	// the original file on disk.
	Hashed map[string]string `json:"hashed"`

	// Integrity maps hashed URL paths to SRI hashes (sha384-...).
	Integrity map[string]string `json:"integrity,omitempty"`
}

// Build scans the given directories for .js files and builds the
// import map. The bare specifier is the filename without the .js
// extension (e.g., "prosemirror-model.js" → "prosemirror-model").
// Each file is content-hashed (SHA-384) for cache busting.
//
// If a directory does not exist or contains no .js files, it is
// skipped. Build returns an empty map when no files are found.
func Build(staticFS fs.FS, dirs ...AssetDir) (*Map, error) {
	m := &Map{
		Imports:   make(map[string]string),
		Hashed:    make(map[string]string),
		Integrity: make(map[string]string),
	}

	for _, d := range dirs {
		if err := m.scanDir(staticFS, d.Dir, d.URLPrefix); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// scanDir reads a single directory and populates the map with its
// .js files. If the directory does not exist, it is silently skipped.
func (m *Map) scanDir(staticFS fs.FS, dir, urlPrefix string) error {
	entries, err := fs.ReadDir(staticFS, dir)
	if err != nil {
		return nil // directory missing — skip
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}

		filename := entry.Name()
		specifier := strings.TrimSuffix(filename, ".js")

		data, err := fs.ReadFile(staticFS, path.Join(dir, filename))
		if err != nil {
			return fmt.Errorf("reading %s: %w", filename, err)
		}

		hash := sha512.Sum384(data)
		digest := hex.EncodeToString(hash[:])
		sri := "sha384-" + base64.StdEncoding.EncodeToString(hash[:])
		hashedPath := urlPrefix + specifier + "-" + digest + ".js"

		m.Imports[specifier] = hashedPath
		m.Hashed[urlPrefix+filename] = hashedPath
		m.Integrity[hashedPath] = sri
	}

	return nil
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
	if m.Integrity == nil {
		m.Integrity = make(map[string]string)
	}
	return &m, nil
}

// Load tries to read a pre-built manifest. If the manifest file does
// not exist, it falls back to building from files on disk.
func Load(publicDir string, dirs ...AssetDir) (*Map, error) {
	manifestPath := path.Join(publicDir, manifestFile)
	m, err := LoadManifest(manifestPath)
	if err == nil {
		return m, nil
	}
	return Build(os.DirFS(publicDir), dirs...)
}

// DevSource pairs a filesystem with a directory to scan and a URL
// prefix. Used by BuildDev to assemble an import map from multiple
// source locations without hashing.
type DevSource struct {
	FS        fs.FS
	ScanDir   string
	URLPrefix string
}

// BuildDev builds an import map for development. It scans each source
// for .js files and maps bare specifiers to unhashed URL paths. No
// file content is read, no hashing is performed, and the Hashed and
// Integrity maps are left empty.
func BuildDev(sources ...DevSource) (*Map, error) {
	m := &Map{
		Imports:   make(map[string]string),
		Hashed:    make(map[string]string),
		Integrity: make(map[string]string),
	}

	for _, src := range sources {
		entries, err := fs.ReadDir(src.FS, src.ScanDir)
		if err != nil {
			continue // directory missing — skip
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
				continue
			}

			specifier := strings.TrimSuffix(entry.Name(), ".js")
			m.Imports[specifier] = src.URLPrefix + entry.Name()
		}
	}

	return m, nil
}

// HeadHTML returns the <script type="importmap"> tag for the document
// head. Returns empty string if there are no imports.
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
	b.WriteByte('}')

	if len(m.Integrity) > 0 {
		ikeys := make([]string, 0, len(m.Integrity))
		for k := range m.Integrity {
			ikeys = append(ikeys, k)
		}
		sort.Strings(ikeys)

		b.WriteString(`,"integrity":{`)
		for i, k := range ikeys {
			if i > 0 {
				b.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			vb, _ := json.Marshal(m.Integrity[k])
			b.Write(kb)
			b.WriteByte(':')
			b.Write(vb)
		}
		b.WriteByte('}')
	}

	b.WriteString(`}</script>`)

	return template.HTML(b.String())
}
