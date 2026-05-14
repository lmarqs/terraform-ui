package sdktest

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStripANSI_WhenInputContainsEscapeCodes_ShouldRemoveThem(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ShouldRemoveColorCodes", "\033[31mhello\033[0m", "hello"},
		{"ShouldRemoveBoldCodes", "\033[1mbold\033[0m", "bold"},
		{"ShouldRemoveMultipleCodes", "\033[32mgreen\033[0m and \033[34mblue\033[0m", "green and blue"},
		{"ShouldHandleNestedCodes", "\033[1;31;42mbold red on green\033[0m", "bold red on green"},
		{"ShouldReturnPlainTextUnchanged", "plain text", "plain text"},
		{"ShouldHandleEmptyString", "", ""},
		{"ShouldRemove256ColorCodes", "\033[38;5;196mred\033[0m", "red"},
		{"ShouldRemoveTrueColorCodes", "\033[38;2;255;0;0mred\033[0m", "red"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNormalizeWhitespace_WhenInputHasTrailingSpaces_ShouldTrimThem(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ShouldTrimTrailingSpaces", "hello   \nworld  \n", "hello\nworld\n"},
		{"ShouldTrimTrailingTabs", "hello\t\nworld\t\n", "hello\nworld\n"},
		{"ShouldTrimTrailingNewlines", "hello\nworld\n\n\n", "hello\nworld\n"},
		{"ShouldPreserveLeadingSpaces", "  hello\n  world\n", "  hello\n  world\n"},
		{"ShouldHandleSingleLine", "hello", "hello\n"},
		{"ShouldHandleEmptyString", "", "\n"},
		{"ShouldHandleMixedLineEndings", "a \n b\t\n c  \n", "a\n b\n c\n"},
		{"ShouldEndWithExactlyOneNewline", "hello\nworld", "hello\nworld\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeWhitespace(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGoldenPath_WhenTestHasSubtests_ShouldReplaceSlashesAndSpaces(t *testing.T) {
	path := goldenPath(t)
	expected := filepath.Join("testdata", "golden", "TestGoldenPath_WhenTestHasSubtests_ShouldReplaceSlashesAndSpaces.txt")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}

	t.Run("SubTest/With/Slashes", func(t *testing.T) {
		path := goldenPath(t)
		expected := filepath.Join("testdata", "golden", "TestGoldenPath_WhenTestHasSubtests_ShouldReplaceSlashesAndSpaces__SubTest__With__Slashes.txt")
		if path != expected {
			t.Fatalf("expected %q, got %q", expected, path)
		}
	})

	t.Run("SubTest With Spaces", func(t *testing.T) {
		path := goldenPath(t)
		expected := filepath.Join("testdata", "golden", "TestGoldenPath_WhenTestHasSubtests_ShouldReplaceSlashesAndSpaces__SubTest_With_Spaces.txt")
		if path != expected {
			t.Fatalf("expected %q, got %q", expected, path)
		}
	})
}

func TestAssertGolden_WhenGoldenFileMatches_ShouldPass(t *testing.T) {
	dir := t.TempDir()
	goldenDir := filepath.Join(dir, "testdata", "golden")
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	content := "hello world\n"
	goldenFile := filepath.Join(goldenDir, "TestAssertGolden_WhenGoldenFileMatches_ShouldPass__ShouldNotFail.txt")
	if err := os.WriteFile(goldenFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}

	t.Run("ShouldNotFail", func(t *testing.T) {
		AssertGolden(t, "hello world")
	})
}

func TestAssertGolden_WhenGoldenFileMissing_ShouldFail(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	mockT := &testing.T{}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		AssertGolden(mockT, "hello")
	}()
	wg.Wait()
	if !mockT.Failed() {
		t.Fatal("expected test to fail when golden file is missing")
	}
}

func TestAssertGolden_WhenGoldenFileMismatches_ShouldFail(t *testing.T) {
	dir := t.TempDir()
	goldenDir := filepath.Join(dir, "testdata", "golden")
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	goldenFile := filepath.Join(goldenDir, ".txt")
	if err := os.WriteFile(goldenFile, []byte("expected content\n"), 0o644); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}

	mockT := &testing.T{}
	AssertGolden(mockT, "different content")
	if !mockT.Failed() {
		t.Fatal("expected test to fail when golden file content mismatches")
	}
}

func TestAssertGolden_WhenInputHasANSI_ShouldStripBeforeComparing(t *testing.T) {
	dir := t.TempDir()
	goldenDir := filepath.Join(dir, "testdata", "golden")
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	content := "hello world\n"
	goldenFile := filepath.Join(goldenDir, "TestAssertGolden_WhenInputHasANSI_ShouldStripBeforeComparing__ShouldMatch.txt")
	if err := os.WriteFile(goldenFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}

	t.Run("ShouldMatch", func(t *testing.T) {
		AssertGolden(t, "\033[31mhello\033[0m world")
	})
}

func TestAssertGolden_WhenUpdateFlagIsSet_ShouldWriteGoldenFile(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	oldUpdate := *update
	*update = true
	t.Cleanup(func() { *update = oldUpdate })

	t.Run("ShouldCreateFile", func(t *testing.T) {
		AssertGolden(t, "created content")

		path := goldenPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected golden file to be created at %s: %v", path, err)
		}
		expected := "created content\n"
		if string(data) != expected {
			t.Fatalf("expected golden file content %q, got %q", expected, string(data))
		}
	})

	t.Run("ShouldCreateDirectoryIfMissing", func(t *testing.T) {
		AssertGolden(t, "new content")

		path := goldenPath(t)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected golden file to exist at %s", path)
		}
	})

	t.Run("ShouldStripANSIBeforeWriting", func(t *testing.T) {
		AssertGolden(t, "\033[31mcolored\033[0m text")

		path := goldenPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected golden file to be created: %v", err)
		}
		expected := "colored text\n"
		if string(data) != expected {
			t.Fatalf("expected golden file content %q, got %q", expected, string(data))
		}
	})

	t.Run("ShouldNormalizeWhitespaceBeforeWriting", func(t *testing.T) {
		AssertGolden(t, "hello   \nworld  \n\n")

		path := goldenPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected golden file to be created: %v", err)
		}
		expected := "hello\nworld\n"
		if string(data) != expected {
			t.Fatalf("expected golden file content %q, got %q", expected, string(data))
		}
	})
}

func TestAssertGolden_WhenInputHasTrailingWhitespace_ShouldNormalizeBeforeComparing(t *testing.T) {
	dir := t.TempDir()
	goldenDir := filepath.Join(dir, "testdata", "golden")
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatalf("failed to create golden dir: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	content := "hello\nworld\n"
	goldenFile := filepath.Join(goldenDir, "TestAssertGolden_WhenInputHasTrailingWhitespace_ShouldNormalizeBeforeComparing__ShouldMatch.txt")
	if err := os.WriteFile(goldenFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}

	t.Run("ShouldMatch", func(t *testing.T) {
		AssertGolden(t, "hello   \nworld  \n\n")
	})
}
