// Package riskpolicy loads complete policy snapshots without exposing source text to plugins.
package riskpolicy

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Module contains exact source bytes under a snapshot-local logical name.
type Module struct {
	Name   string
	Source string
}

// Snapshot owns immutable module bytes and their report identity.
type Snapshot struct {
	kind    string
	digest  string
	modules []Module
}

func (s Snapshot) Kind() string   { return s.kind }
func (s Snapshot) Digest() string { return s.digest }
func (s Snapshot) ID() string {
	if s.kind == "builtin" {
		return "builtin/v1"
	}
	return "custom/" + s.digest
}
func (s Snapshot) Modules() []Module { return slices.Clone(s.modules) }

// LoadSource captures a selected complete source. The evaluator supplies its embedded
// modules for built-in selection; custom selection never mixes them into the snapshot.
func LoadSource(selection config.RiskPolicySelection, builtin []Module) (Snapshot, error) {
	return (sourceReader{os.Lstat, os.ReadFile, os.ReadDir}).load(selection, builtin)
}

// Separate filesystem operations permit deterministic checks of observed edits and
// read failures without relying on scheduler timing or changing global functions.
type sourceReader struct {
	lstat    func(string) (os.FileInfo, error)
	readFile func(string) ([]byte, error)
	readDir  func(string) ([]os.DirEntry, error)
}

func (r sourceReader) load(selection config.RiskPolicySelection, builtin []Module) (Snapshot, error) {
	kind := "custom"
	var modules []Module
	var err error
	if selection.Path == "" {
		kind = "builtin"
		modules = slices.Clone(builtin)
	} else {
		modules, err = r.readSource(selection.Path)
		if err != nil {
			return Snapshot{}, err
		}
		verified, err := r.readSource(selection.Path)
		if err != nil || !slices.Equal(modules, verified) {
			return Snapshot{}, sdk.ErrorSourceChanged
		}
	}
	slices.SortFunc(modules, func(a, b Module) int { return strings.Compare(a.Name, b.Name) })
	if len(modules) == 0 {
		return Snapshot{}, sdk.ErrorSourceInvalid
	}
	for i, module := range modules {
		if !utf8.ValidString(module.Name) || !productionModule(module.Name) || strings.TrimSpace(module.Source) == "" || (i > 0 && modules[i-1].Name == module.Name) {
			return Snapshot{}, sdk.ErrorSourceInvalid
		}
	}
	return Snapshot{kind: kind, digest: sourceDigest(modules), modules: modules}, nil
}

func sourceDigest(modules []Module) string {
	hash := sha256.New()
	hash.Write([]byte("tfui-risk-source-v1\x00"))
	var size [8]byte
	writeSize := func(n int) { binary.BigEndian.PutUint64(size[:], uint64(n)); hash.Write(size[:]) }
	writeSize(len(modules))
	for _, module := range modules {
		writeSize(len(module.Name))
		hash.Write([]byte(module.Name))
		writeSize(len(module.Source))
		hash.Write([]byte(module.Source))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func productionModule(name string) bool {
	return strings.HasSuffix(name, ".rego") && !strings.HasSuffix(name, "_test.rego")
}

func (r sourceReader) readSource(path string) ([]Module, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, sdk.ErrorSourceUnreadable
	}
	if err := r.rejectSymlinks(path); err != nil {
		return nil, err
	}
	info, err := r.lstat(path)
	if err != nil {
		return nil, sdk.ErrorSourceUnreadable
	}
	if info.Mode().IsRegular() {
		if !productionModule(filepath.Base(path)) {
			return nil, sdk.ErrorSourceInvalid
		}
		module, err := r.readModule(path)
		if err != nil {
			return nil, err
		}
		return []Module{module}, nil
	}
	if !info.IsDir() {
		return nil, sdk.ErrorSourceInvalid
	}
	entries, err := r.readDir(path)
	if err != nil {
		return nil, sdk.ErrorSourceUnreadable
	}
	var modules []Module
	for _, entry := range entries {
		if !productionModule(entry.Name()) || entry.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, sdk.ErrorSourceInvalid
		}
		module, err := r.readModule(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func (r sourceReader) readModule(path string) (Module, error) {
	info, err := r.lstat(path)
	if err != nil {
		return Module{}, sdk.ErrorSourceUnreadable
	}
	if !info.Mode().IsRegular() {
		return Module{}, sdk.ErrorSourceInvalid
	}
	data, err := r.readFile(path)
	if err != nil {
		return Module{}, sdk.ErrorSourceUnreadable
	}
	return Module{Name: filepath.Base(path), Source: string(data)}, nil
}

func (r sourceReader) rejectSymlinks(path string) error {
	for {
		info, err := r.lstat(path)
		if err != nil {
			return sdk.ErrorSourceUnreadable
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return sdk.ErrorSourceInvalid
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}
