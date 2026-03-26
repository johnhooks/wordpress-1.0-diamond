package parse

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	fixtures, err := filepath.Glob("testdata/*.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found in testdata/")
	}

	for _, fixture := range fixtures {
		name := strings.TrimSuffix(filepath.Base(fixture), ".html")
		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := ParseTemplate(strings.NewReader(string(input)))
			if err != nil {
				t.Fatalf("ParseTemplate error: %v", err)
			}

			got := Sprint(doc)
			goldenPath := fixture[:len(fixture)-len(".html")] + ".golden"

			if *update {
				if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file (run with -update to create): %v", err)
			}

			if got != string(want) {
				t.Errorf("AST mismatch for %s\n\ngot:\n%s\nwant:\n%s", name, got, string(want))
			}
		})
	}
}
