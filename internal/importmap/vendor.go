package importmap

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const esmBase = "https://esm.sh"

// depLine matches esm.sh barrel file dependency imports:
//
//	import "/{name}@{version}?target=es2022";
var depLine = regexp.MustCompile(`import\s+"/([\w@][\w.\-/]*)@([^"?]+)`)

// commentVersion matches the resolved version in esm.sh's comment:
//
//	/* esm.sh - {name}@{version} */
var commentVersion = regexp.MustCompile(`/\*\s*esm\.sh\s*-\s*([\w@][\w.\-/]*)@(\S+)\s*\*/`)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Vendor fetches all packages declared in pins and their transitive
// dependencies from esm.sh, saves them to the vendor directory, and
// builds the manifest. The resolved pins (with discovered transitive
// deps) are written back to pinsPath.
//
// pins maps bare specifiers to versions (e.g., "prosemirror-model": "1.24.1").
// Transitive dependencies are discovered automatically.
func Vendor(log io.Writer, publicDir, vendorDir, urlPrefix, pinsPath string, pins map[string]string) (*Map, error) {
	absVendor := filepath.Join(publicDir, vendorDir)
	if err := os.MkdirAll(absVendor, 0755); err != nil {
		return nil, fmt.Errorf("creating vendor dir: %w", err)
	}

	// resolved tracks every package we need: name → exact version.
	// Seeded from explicit pins, extended by dependency discovery.
	resolved := make(map[string]string, len(pins))
	for name, version := range pins {
		resolved[name] = version
	}

	// Phase 1: Discover transitive dependencies by fetching barrel
	// files from esm.sh (without * prefix). Each barrel lists deps
	// with semver ranges. We resolve exact versions by fetching each
	// dep's barrel and reading the comment header.
	visited := make(map[string]bool)
	queue := make([]string, 0, len(pins))
	for name := range pins {
		queue = append(queue, name)
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if visited[name] {
			continue
		}
		visited[name] = true

		version := resolved[name]
		fmt.Fprintf(log, "  resolve %s@%s\n", name, version)
		barrel, err := fetchString(fmt.Sprintf("%s/%s@%s?target=es2022", esmBase, name, version))
		if err != nil {
			return nil, fmt.Errorf("fetching barrel for %s@%s: %w", name, version, err)
		}

		// If the version was a range, the barrel comment has the
		// resolved version.
		if m := commentVersion.FindStringSubmatch(barrel); m != nil {
			resolved[name] = m[2]
		}

		// Discover dependencies.
		for _, m := range depLine.FindAllStringSubmatch(barrel, -1) {
			depName := m[1]
			depRange := m[2]

			if _, known := resolved[depName]; known {
				continue
			}

			// Resolve the semver range to an exact version.
			depBarrel, err := fetchString(fmt.Sprintf("%s/%s@%s?target=es2022", esmBase, depName, depRange))
			if err != nil {
				return nil, fmt.Errorf("resolving %s@%s: %w", depName, depRange, err)
			}
			cm := commentVersion.FindStringSubmatch(depBarrel)
			if cm == nil {
				return nil, fmt.Errorf("could not resolve version for %s@%s", depName, depRange)
			}
			resolved[depName] = cm[2]
			fmt.Fprintf(log, "  found   %s@%s (from %s)\n", depName, cm[2], name)
			queue = append(queue, depName)
		}
	}

	// Phase 2: Fetch the actual module code with bare specifiers
	// (using the * prefix) and save to vendor directory.
	fetched := 0
	for name, version := range resolved {
		fetched++
		fmt.Fprintf(log, "  fetch   %s@%s (%d/%d)\n", name, version, fetched, len(resolved))
		url := fmt.Sprintf("%s/*%s@%s/es2022/%s.mjs", esmBase, name, version, name)
		code, err := fetchString(url)
		if err != nil {
			return nil, fmt.Errorf("fetching %s@%s: %w", name, version, err)
		}

		dest := filepath.Join(absVendor, name+".js")
		if err := os.WriteFile(dest, []byte(code), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", dest, err)
		}
	}

	// Phase 3: Build manifest from the fetched files.
	m, err := Build(os.DirFS(publicDir), vendorDir, urlPrefix)
	if err != nil {
		return nil, fmt.Errorf("building manifest: %w", err)
	}

	manifestPath := filepath.Join(absVendor, manifestFile)
	if err := m.WriteManifest(manifestPath); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	// Write resolved pins (with discovered transitive deps) back.
	if err := writePins(pinsPath, resolved); err != nil {
		return nil, fmt.Errorf("writing pins: %w", err)
	}

	return m, nil
}

// writePins writes the resolved pins as a flat JSON map.
func writePins(path string, resolved map[string]string) error {
	data, err := marshalSorted(resolved)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// marshalSorted produces indented JSON with sorted keys.
func marshalSorted(m map[string]string) ([]byte, error) {
	keys := sortedKeys(m)
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "  %q: %q", k, m[k])
	}
	b.WriteString("\n}\n")
	return []byte(b.String()), nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fetchString(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
