package assets

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"press/internal/importmap"
	"press/js"
)

// Pipeline configures the asset build for a Press project. It holds
// the vendor dependency list, the embedded engine JS, the theme
// directory, and the asset directories to scan for the import map.
type Pipeline struct {
	// Deps maps bare specifiers to esm.sh versions. Seeded with
	// DefaultDeps by Default() and extended via AddDeps().
	Deps map[string]string

	// EngineJS is the embedded filesystem containing engine-authored
	// JavaScript files. These are written to publicDir/js/ by Run.
	EngineJS fs.FS

	// ThemeDir is the active theme directory. If it contains a js/
	// subdirectory, those files are copied to publicDir/theme/.
	ThemeDir string

	// AssetDirs lists the directories to scan within publicDir when
	// building the import map manifest.
	AssetDirs []importmap.AssetDir
}

// Default returns a Pipeline configured with the Press engine's
// vendor dependencies, embedded JS, and standard asset directories.
func Default() *Pipeline {
	deps := make(map[string]string, len(DefaultDeps))
	for k, v := range DefaultDeps {
		deps[k] = v
	}
	return &Pipeline{
		Deps:     deps,
		EngineJS: js.FS,
		AssetDirs: []importmap.AssetDir{
			{Dir: "vendor", URLPrefix: "/static/vendor/"},
			{Dir: "js", URLPrefix: "/static/js/"},
			{Dir: "theme", URLPrefix: "/static/theme/"},
		},
	}
}

// AddDeps merges additional esm.sh dependencies into the pipeline.
func (p *Pipeline) AddDeps(deps map[string]string) {
	for k, v := range deps {
		p.Deps[k] = v
	}
}

// AddAssetDir appends an additional directory to scan when building
// the import map.
func (p *Pipeline) AddAssetDir(dir importmap.AssetDir) {
	p.AssetDirs = append(p.AssetDirs, dir)
}

// Run executes the asset pipeline:
//  1. Writes embedded engine JS to publicDir/js/
//  2. Copies theme JS from ThemeDir/js/ to publicDir/theme/ (if present)
//  3. Fetches vendor deps from esm.sh
//  4. Builds the import map manifest covering all asset directories
func (p *Pipeline) Run(log io.Writer, publicDir string) (*importmap.Map, error) {
	// Step 1: Write engine JS from embedded FS.
	if p.EngineJS != nil {
		destDir := filepath.Join(publicDir, "js")
		if err := writeFS(log, p.EngineJS, destDir); err != nil {
			return nil, fmt.Errorf("writing engine JS: %w", err)
		}
	}

	// Step 2: Copy theme JS.
	if p.ThemeDir != "" {
		themeJS := filepath.Join(p.ThemeDir, "js")
		if info, err := os.Stat(themeJS); err == nil && info.IsDir() {
			destDir := filepath.Join(publicDir, "theme")
			if err := writeFS(log, os.DirFS(themeJS), destDir); err != nil {
				return nil, fmt.Errorf("copying theme JS: %w", err)
			}
		}
	}

	// Step 3: Fetch vendor deps and build manifest.
	m, err := importmap.Vendor(log, publicDir, p.AssetDirs, p.Deps)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// writeFS writes all .js files from an fs.FS to a destination
// directory on disk, creating it if needed.
func writeFS(log io.Writer, fsys fs.FS, dest string) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}

		data, err := fs.ReadFile(fsys, entry.Name())
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}

		fmt.Fprintf(log, "  write   %s\n", entry.Name())
	}

	return nil
}

// CleanVendor removes all .js files from the vendor directory.
func CleanVendor(log io.Writer, publicDir string) error {
	absVendor := filepath.Join(publicDir, "vendor")
	matches, _ := filepath.Glob(filepath.Join(absVendor, "*.js"))
	for _, f := range matches {
		if err := os.Remove(f); err != nil {
			return fmt.Errorf("removing %s: %w", f, err)
		}
		fmt.Fprintf(log, "  removed %s\n", filepath.Base(f))
	}
	return nil
}
