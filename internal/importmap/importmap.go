package importmap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

// Map generates an import map from vendored JavaScript modules. It
// reads files from the vendor directory in the provided filesystem,
// computes content hashes for cache busting, and produces the JSON
// and HTML needed for <script type="importmap">.
type Map struct {
	// Imports maps bare specifiers to hashed URL paths.
	Imports map[string]string

	// Hashed maps original paths to hashed paths for serving.
	Hashed map[string]string
}

// Build scans the vendor directory for .js files and builds the
// import map. File names are mapped to bare specifiers by stripping
// the .js extension (e.g., "prosemirror-model.js" → "prosemirror-model").
// Each file gets a content hash appended to its path for cache busting.
func Build(staticFS fs.FS, vendorDir, urlPrefix string) (*Map, error) {
	m := &Map{
		Imports: make(map[string]string),
		Hashed: make(map[string]string),
	}

	err := fs.WalkDir(staticFS, vendorDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".js") {
			return nil
		}

		data, err := fs.ReadFile(staticFS, p)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(data)
		shortHash := hex.EncodeToString(hash[:8])

		base := path.Base(p)
		name := strings.TrimSuffix(base, ".js")
		ext := ".js"
		hashedName := name + "-" + shortHash + ext
		hashedPath := urlPrefix + hashedName

		m.Imports[name] = hashedPath
		m.Hashed[urlPrefix+base] = hashedPath

		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

// JSON returns the import map as a JSON string suitable for
// embedding in a <script type="importmap"> tag.
func (m *Map) JSON() (string, error) {
	obj := struct {
		Imports map[string]string `json:"imports"`
	}{Imports: m.Imports}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Tag returns the complete <script type="importmap"> HTML tag.
func (m *Map) Tag() (template.HTML, error) {
	j, err := m.JSON()
	if err != nil {
		return "", err
	}
	return template.HTML(`<script type="importmap">` + j + `</script>`), nil
}
