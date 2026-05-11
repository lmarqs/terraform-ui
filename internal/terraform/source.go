package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/editor"
)

// SourceIndex maps resource addresses to their definition locations in .tf files.
type SourceIndex struct {
	dir       string
	locations map[string]editor.SourceLocation
}

// NewSourceIndex builds an index by scanning all .tf files in the given directory
// (and subdirectories) for resource, data, and module block definitions.
func NewSourceIndex(dir string) (*SourceIndex, error) {
	idx := &SourceIndex{
		dir:       dir,
		locations: make(map[string]editor.SourceLocation),
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".terraform" || base == ".git" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".tf" || filepath.Ext(path) == ".tofu" {
			idx.scanFile(path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dir, err)
	}

	return idx, nil
}

// Lookup returns the source location for a resource address.
// Walks up the address tree trying progressively less specific lookups:
// 1. Exact match
// 2. Strip module prefix → leaf resource
// 3. Strip index suffixes → base resource
// 4. Walk up module hierarchy → find the module call declaration
func (idx *SourceIndex) Lookup(address string) (editor.SourceLocation, bool) {
	if loc, ok := idx.locations[address]; ok {
		return loc, true
	}
	// Strip module prefix: "module.foo.module.bar.aws_instance.x" → "aws_instance.x"
	leaf := stripModulePrefix(address)
	if leaf != address {
		if loc, ok := idx.locations[leaf]; ok {
			return loc, true
		}
	}
	// Strip index suffixes: "aws_instance.web[0]" → "aws_instance.web"
	bare := stripIndex(leaf)
	if bare != leaf {
		if loc, ok := idx.locations[bare]; ok {
			return loc, true
		}
	}
	// Walk up module hierarchy: try to find the module call declaration
	if loc, ok := idx.lookupModuleCall(address); ok {
		return loc, true
	}
	return editor.SourceLocation{}, false
}

// lookupModuleCall walks up the module hierarchy trying to find where
// the module is declared. For "module.a.module.b.aws_instance.x", tries:
// "module.a.module.b", then "module.a".
func (idx *SourceIndex) lookupModuleCall(address string) (editor.SourceLocation, bool) {
	remaining := address
	for strings.HasPrefix(remaining, "module.") {
		rest := remaining[len("module."):]
		dot := dotAfterSegment(rest)
		if dot < 0 {
			break
		}
		modulePath := remaining[:len("module.")+dot]
		bare := stripIndex(modulePath)
		if loc, ok := idx.locations[bare]; ok {
			return loc, true
		}
		if bare != modulePath {
			if loc, ok := idx.locations[modulePath]; ok {
				return loc, true
			}
		}
		remaining = rest[dot+1:]
	}
	return editor.SourceLocation{}, false
}

// stripIndex removes bracket index suffixes from each segment of an address.
// "aws_iam_user.this[0]" → "aws_iam_user.this"
func stripIndex(address string) string {
	idx := strings.Index(address, "[")
	if idx < 0 {
		return address
	}
	return address[:idx]
}

// stripModulePrefix removes all "module.name." prefixes, respecting brackets.
func stripModulePrefix(address string) string {
	for strings.HasPrefix(address, "module.") {
		rest := address[len("module."):]
		dot := dotAfterSegment(rest)
		if dot < 0 {
			return address
		}
		address = rest[dot+1:]
	}
	return address
}

// dotAfterSegment finds the first '.' that is not inside brackets.
func dotAfterSegment(s string) int {
	depth := 0
	for i, ch := range s {
		switch ch {
		case '[':
			depth++
		case ']':
			depth--
		case '.':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// LookupFile returns the directory's main.tf (or first .tf file) as a fallback.
func (idx *SourceIndex) LookupFile(dir string) (editor.SourceLocation, bool) {
	mainTf := filepath.Join(dir, "main.tf")
	if _, err := os.Stat(mainTf); err == nil {
		return editor.SourceLocation{File: mainTf, Line: 1}, true
	}
	// Fallback: first .tf file in directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return editor.SourceLocation{}, false
	}
	for _, e := range entries {
		if !e.IsDir() && (filepath.Ext(e.Name()) == ".tf" || filepath.Ext(e.Name()) == ".tofu") {
			return editor.SourceLocation{File: filepath.Join(dir, e.Name()), Line: 1}, true
		}
	}
	return editor.SourceLocation{}, false
}

// Count returns the number of indexed resource locations.
func (idx *SourceIndex) Count() int {
	return len(idx.locations)
}

// resource/data block regex patterns
var (
	resourceBlockRe = regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"`)
	dataBlockRe     = regexp.MustCompile(`^\s*data\s+"([^"]+)"\s+"([^"]+)"`)
	moduleBlockRe   = regexp.MustCompile(`^\s*module\s+"([^"]+)"`)
)

func (idx *SourceIndex) scanFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Determine the module prefix from the file path relative to the root dir
	modulePrefix := idx.modulePrefix(path)

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		lineNum := i + 1

		// Match resource blocks: resource "type" "name" {
		if matches := resourceBlockRe.FindStringSubmatch(line); matches != nil {
			resourceType := matches[1]
			resourceName := matches[2]
			address := resourceType + "." + resourceName
			if modulePrefix != "" {
				address = modulePrefix + "." + address
			}
			idx.locations[address] = editor.SourceLocation{
				File: path,
				Line: lineNum,
				Col:  1,
			}
			continue
		}

		// Match data blocks: data "type" "name" {
		if matches := dataBlockRe.FindStringSubmatch(line); matches != nil {
			dataType := matches[1]
			dataName := matches[2]
			address := "data." + dataType + "." + dataName
			if modulePrefix != "" {
				address = modulePrefix + "." + address
			}
			idx.locations[address] = editor.SourceLocation{
				File: path,
				Line: lineNum,
				Col:  1,
			}
			continue
		}

		// Match module blocks: module "name" {
		if matches := moduleBlockRe.FindStringSubmatch(line); matches != nil {
			moduleName := matches[1]
			address := "module." + moduleName
			if modulePrefix != "" {
				address = modulePrefix + "." + address
			}
			idx.locations[address] = editor.SourceLocation{
				File: path,
				Line: lineNum,
				Col:  1,
			}
			continue
		}
	}
}

// modulePrefix determines the module path prefix for a file based on its directory
// relative to the root. Returns empty for root-level files.
func (idx *SourceIndex) modulePrefix(path string) string {
	dir := filepath.Dir(path)
	if dir == idx.dir {
		return ""
	}

	rel, err := filepath.Rel(idx.dir, dir)
	if err != nil {
		return ""
	}

	// Only generate module prefix for known module directories
	// (directories inside modules/ or similar structures)
	// For simple cases, return empty — the resource address in state
	// already contains the full module path.
	_ = rel
	return ""
}
