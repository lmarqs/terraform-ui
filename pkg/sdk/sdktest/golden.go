package sdktest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var update = flag.Bool("update", false, "update golden files")

func StripANSI(s string) string {
	return ansi.Strip(s)
}

func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	result := strings.Join(lines, "\n")
	return strings.TrimRight(result, "\n") + "\n"
}

func AssertGolden(t *testing.T, actual string) {
	t.Helper()
	stripped := normalizeWhitespace(StripANSI(actual))
	path := goldenPath(t)

	if *update {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(stripped), 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file not found: %s\nRun with -update to create it", path)
	}

	if stripped != string(expected) {
		t.Errorf("output does not match golden file %s:\n%s", path, unifiedDiff(string(expected), stripped))
	}
}

func goldenPath(t *testing.T) string {
	name := t.Name()
	name = strings.ReplaceAll(name, "/", "__")
	name = strings.ReplaceAll(name, " ", "_")
	return filepath.Join("testdata", "golden", name+".txt")
}
